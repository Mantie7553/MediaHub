"""
First-run wiring for the MediaHub Docker stack. Standard library only.

  bootstrap.py qbit-config   write qBittorrent's config before its first start (password, save paths)
  bootstrap.py               create the media folders and connect Sonarr, Radarr, Prowlarr,
                             qBittorrent and MediaHub through their APIs

Every step checks what already exists first, so running it again changes nothing.
"""

import base64
import hashlib
import http.cookiejar
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

MEDIA_ROOT = os.environ.get("MEDIA_ROOT", "/media").rstrip("/")
ARR_USER = "admin"
ARR_PASSWORD = os.environ.get("ARR_PASSWORD", "")
WEBHOOK_SECRET = os.environ.get("WEBHOOK_SECRET", "")
PUID = int(os.environ.get("PUID", "1000"))
PGID = int(os.environ.get("PGID", "1000"))

# folders MediaHub expects under MEDIA_ROOT (see architecture.md), plus qBittorrent's downloads
MEDIA_FOLDERS = ["TV Shows", "Movies", "Music", "Manga", "Light Novels", "temp", "downloads"]
DOWNLOADS = f"{MEDIA_ROOT}/downloads"

QBIT = {"host": "qbittorrent", "port": 8080}
BACKEND_URL = "http://backend:9090"
SETTINGS_FILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "arr-settings.json")

# stop torrents the moment they finish (ratio 0, no seeding time), Sonarr/Radarr then import and
# remove them. max_ratio_act 0 is "stop"
QBIT_NO_SEEDING = {
    "max_ratio_enabled": True,
    "max_ratio": 0,
    "max_seeding_time_enabled": True,
    "max_seeding_time": 0,
    "max_ratio_act": 0,
}


def log(msg):
    print(msg, flush=True)


# ---------------------------------------------------------------- qBittorrent config

def qbit_password_hash(password):
    """qBittorrent's WebUI\\Password_PBKDF2 value: PBKDF2-SHA512, 100000 rounds, 16 byte salt."""
    salt = os.urandom(16)
    key = hashlib.pbkdf2_hmac("sha512", password.encode(), salt, 100000, dklen=64)
    return f"@ByteArray({base64.b64encode(salt).decode()}:{base64.b64encode(key).decode()})"


def qbit_config():
    path = "/config/qBittorrent/qBittorrent.conf"
    if os.path.exists(path):
        log("qBittorrent config already exists, leaving it alone")
        return
    if not ARR_PASSWORD:
        sys.exit("ARR_PASSWORD is not set")
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", newline="\n") as f:
        f.write(
            "[BitTorrent]\n"
            f"Session\\DefaultSavePath={DOWNLOADS}\n"
            f"Session\\TempPath={DOWNLOADS}/incomplete\n"
            "Session\\TempPathEnabled=true\n"
            "\n"
            "[LegalNotice]\n"
            "Accepted=true\n"
            "\n"
            "[Preferences]\n"
            f"WebUI\\Username={ARR_USER}\n"
            f'WebUI\\Password_PBKDF2="{qbit_password_hash(ARR_PASSWORD)}"\n'
            "WebUI\\Port=8080\n"
        )
    log("wrote qBittorrent config")


# ---------------------------------------------------------------- HTTP helpers

class Api:
    def __init__(self, name, base, version, key):
        self.name = name
        self.base = f"{base}/api/{version}"
        self.root = base
        self.key = key

    def call(self, method, path, body=None):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(self.base + path, data=data, method=method)
        req.add_header("X-Api-Key", self.key)
        req.add_header("Content-Type", "application/json")
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                raw = resp.read()
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as e:
            detail = e.read().decode(errors="replace")[:500]
            raise RuntimeError(f"{self.name} {method} {path} -> {e.code}: {detail}") from None

    def get(self, path):
        return self.call("GET", path)

    def post(self, path, body):
        return self.call("POST", path, body)

    def put(self, path, body):
        return self.call("PUT", path, body)

    def wait(self, timeout=300):
        """Wait until the app answers an authenticated request, so the API key is in effect."""
        deadline = time.time() + timeout
        last = None
        while time.time() < deadline:
            try:
                self.get("/system/status")
                log(f"{self.name} is up")
                return
            except Exception as e:  # not started yet, or still migrating its database
                last = e
                time.sleep(3)
        raise RuntimeError(f"{self.name} did not come up: {last}")


def wait_for_url(name, url, timeout=300):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            urllib.request.urlopen(url, timeout=5).close()
            log(f"{name} is up")
            return
        except urllib.error.HTTPError:
            log(f"{name} is up")  # any HTTP answer means the server is running
            return
        except Exception:
            time.sleep(3)
    raise RuntimeError(f"{name} did not come up at {url}")


def from_schema(api, kind, implementation, name, values, extra=None):
    """Build a provider (download client, notification, application) from the app's own schema,
    so fields we don't set keep the app's defaults."""
    schema = next(s for s in api.get(f"/{kind}/schema") if s["implementation"] == implementation)
    item = dict(schema)
    item.pop("id", None)
    item["name"] = name
    item["enable"] = True
    for field in item["fields"]:
        if field["name"] in values:
            field["value"] = values[field["name"]]
    missing = set(values) - {f["name"] for f in item["fields"]}
    if missing:
        log(f"  warning: {api.name} {implementation} has no fields {sorted(missing)}")
    item.update(extra or {})
    return item


def ensure_provider(api, kind, implementation, name, values, extra=None):
    existing = [p for p in api.get(f"/{kind}") if p["implementation"] == implementation]
    if any(p["name"] == name for p in existing):
        log(f"  {api.name}: {kind} '{name}' already set up")
        return
    api.post(f"/{kind}", from_schema(api, kind, implementation, name, values, extra))
    log(f"  {api.name}: added {kind} '{name}'")


# ---------------------------------------------------------------- qBittorrent API

def setup_qbit():
    """Turn seeding off. Applied on every run, so it also fixes an install where it was changed."""
    log("qBittorrent:")
    base = f"http://{QBIT['host']}:{QBIT['port']}"
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def call(path, form=None):
        data = urllib.parse.urlencode(form).encode() if form is not None else None
        req = urllib.request.Request(base + path, data=data, headers={"Referer": base})
        with opener.open(req, timeout=30) as resp:
            return resp.read().decode()

    if call("/api/v2/auth/login", {"username": ARR_USER, "password": ARR_PASSWORD}).strip() == "Fails.":
        raise RuntimeError("qBittorrent rejected the login, is ARR_PASSWORD the one it was set up with?")
    prefs = json.loads(call("/api/v2/app/preferences"))
    if all(prefs.get(k) == v for k, v in QBIT_NO_SEEDING.items()):
        log("  qBittorrent: seeding already off")
        return
    call("/api/v2/app/setPreferences", {"json": json.dumps(QBIT_NO_SEEDING)})
    log("  qBittorrent: seeding turned off (stop at ratio 0)")


# ---------------------------------------------------------------- setup steps

def make_media_folders():
    for folder in MEDIA_FOLDERS:
        path = os.path.join(MEDIA_ROOT, folder)
        if not os.path.isdir(path):
            os.makedirs(path, exist_ok=True)
            log(f"created {path}")
        try:
            os.chown(path, PUID, PGID)
        except OSError:
            pass  # network shares and Windows mounts don't support chown, ownership comes from the mount


def set_login(api):
    """Turn on forms login with the shared admin password. A fresh *arr install otherwise
    shows an 'authentication required' setup screen."""
    host = api.get("/config/host")
    if host.get("authenticationMethod") == "forms" and host.get("username"):
        log(f"  {api.name}: login already configured")
        return
    host.update({
        "authenticationMethod": "forms",
        "authenticationRequired": "enabled",
        "username": ARR_USER,
        "password": ARR_PASSWORD,
        "passwordConfirmation": ARR_PASSWORD,
    })
    api.put(f"/config/host/{host['id']}", host)
    log(f"  {api.name}: login set to {ARR_USER}")


def ensure_root_folder(api, path):
    if any(r["path"].rstrip("/") == path for r in api.get("/rootfolder")):
        log(f"  {api.name}: root folder {path} already set up")
        return
    api.post("/rootfolder", {"path": path})
    log(f"  {api.name}: added root folder {path}")


def cf_signature(cf):
    """Comparable form of a custom format, from the file or from the API."""
    specs = []
    for spec in cf["specifications"]:
        fields = spec["fields"]
        if isinstance(fields, list):
            fields = {f["name"]: f.get("value") for f in fields}
        specs.append((spec["name"], spec["implementation"], bool(spec.get("negate", False)),
                      bool(spec.get("required", False)), fields.get("value")))
    return sorted(specs)


def sync_custom_formats(api, formats):
    existing = {cf["name"]: cf for cf in api.get("/customformat")}
    schema = {s["implementation"]: s for s in api.get("/customformat/schema")}
    for cf in formats:
        specs = []
        for spec in cf["specifications"]:
            item = json.loads(json.dumps(schema[spec["implementation"]]))
            item["name"] = spec["name"]
            item["negate"] = spec.get("negate", False)
            item["required"] = spec.get("required", False)
            for field in item["fields"]:
                if field["name"] in spec["fields"]:
                    field["value"] = spec["fields"][field["name"]]
            specs.append(item)
        body = {"name": cf["name"], "includeCustomFormatWhenRenaming": False, "specifications": specs}

        current = existing.get(cf["name"])
        if current is None:
            api.post("/customformat", body)
            log(f"  {api.name}: added custom format '{cf['name']}'")
        elif cf_signature(current) != cf_signature(body):
            api.put(f"/customformat/{current['id']}", dict(body, id=current["id"]))
            log(f"  {api.name}: updated custom format '{cf['name']}'")
        else:
            log(f"  {api.name}: custom format '{cf['name']}' up to date")


def build_profile(api, want):
    """Quality profile from the file: qualities grouped per resolution, everything else not allowed."""
    profile = api.get("/qualityprofile/schema")
    # flatten the schema into single qualities, and remember which schema names (e.g. the
    # "WEB 1080p" group) stand for which qualities
    singles, by_name = [], {}
    for item in profile["items"]:
        if item.get("quality"):
            singles.append(item["quality"])
            by_name[item["quality"]["name"]] = [item["quality"]]
        else:
            members = [sub["quality"] for sub in item["items"]]
            singles.extend(members)
            by_name[item["name"]] = members
            for q in members:
                by_name[q["name"]] = [q]

    groups, used = [], set()
    for n, group in enumerate(want["groups"]):
        qualities = []
        for name in group["qualities"]:
            if name not in by_name:
                raise RuntimeError(f"{api.name} has no quality named '{name}'")
            qualities.extend(by_name[name])
        used.update(q["id"] for q in qualities)
        groups.append({
            "id": 1000 + n,
            "name": group["name"],
            "allowed": True,
            "items": [{"quality": q, "items": [], "allowed": True} for q in qualities],
        })
    # the API lists the lowest rank first, so unused qualities go before the groups
    items = [{"quality": q, "items": [], "allowed": False} for q in singles if q["id"] not in used] + groups

    cutoff = next((g["id"] for g in groups if g["name"] == want["cutoff"]), None)
    if cutoff is None:
        raise RuntimeError(f"cutoff '{want['cutoff']}' is not one of the groups")

    scores = want.get("formatScores", {})
    formats = {cf["name"]: cf["id"] for cf in api.get("/customformat")}
    missing = set(scores) - set(formats)
    if missing:
        raise RuntimeError(f"{api.name} has no custom formats {sorted(missing)}")

    profile.update({
        "name": want["name"],
        "items": items,
        "cutoff": cutoff,
        "upgradeAllowed": want.get("upgradeAllowed", False),
        "minFormatScore": want.get("minFormatScore", 0),
        "cutoffFormatScore": want.get("cutoffFormatScore", 0),
        "minUpgradeFormatScore": want.get("minUpgradeFormatScore", 1),
        "formatItems": [{"format": fid, "name": name, "score": scores.get(name, 0)} for name, fid in formats.items()],
    })
    if "language" in profile and "radarrLanguage" in want:
        profile["language"] = want["radarrLanguage"]
    return profile


def profile_signature(p):
    def item_sig(i):
        name = i["quality"]["name"] if i.get("quality") else i["name"]
        return (name, i["allowed"], tuple(sorted(sub["quality"]["name"] for sub in i.get("items", []))))
    cutoff_name = next(((i["quality"]["name"] if i.get("quality") else i["name"]) for i in p["items"]
                        if (i.get("quality") or {}).get("id", i.get("id")) == p["cutoff"]), None)
    return (
        [item_sig(i) for i in p["items"] if i["allowed"]],
        cutoff_name,
        p["upgradeAllowed"], p["minFormatScore"], p["cutoffFormatScore"], p.get("minUpgradeFormatScore"),
        sorted((f["name"], f["score"]) for f in p["formatItems"] if f["score"]),
        (p.get("language") or {}).get("id"),
    )


def sync_quality_profiles(api, profiles):
    existing = {p["name"]: p for p in api.get("/qualityprofile")}
    for want in profiles:
        body = build_profile(api, want)
        current = existing.get(want["name"])
        if current is None:
            api.post("/qualityprofile", body)
            log(f"  {api.name}: added quality profile '{want['name']}'")
        elif profile_signature(current) != profile_signature(body):
            api.put(f"/qualityprofile/{current['id']}", dict(body, id=current["id"]))
            log(f"  {api.name}: updated quality profile '{want['name']}'")
        else:
            log(f"  {api.name}: quality profile '{want['name']}' up to date")


def check_quality_profile(api):
    # downloader/dispatch.go adds everything with qualityProfileId 1
    if not any(p["id"] == 1 for p in api.get("/qualityprofile")):
        log(f"  warning: {api.name} has no quality profile with id 1, MediaHub requests will fail")


def setup_arr(api, settings, media_folder, category_field, category, webhook_path):
    log(f"{api.name}:")
    set_login(api)
    ensure_root_folder(api, f"{MEDIA_ROOT}/{media_folder}")
    ensure_provider(api, "downloadclient", "QBittorrent", "qBittorrent", {
        "host": QBIT["host"],
        "port": QBIT["port"],
        "username": ARR_USER,
        "password": ARR_PASSWORD,
        category_field: category,
    }, extra={"removeCompletedDownloads": True, "removeFailedDownloads": True})
    ensure_provider(api, "notification", "Webhook", "MediaHub", {
        "url": BACKEND_URL + webhook_path,
        "method": 1,  # POST
        "headers": [{"key": "X-MediaHub-Secret", "value": WEBHOOK_SECRET}],
    }, extra={"onDownload": True, "onUpgrade": True})
    # formats first, profiles score them by name
    sync_custom_formats(api, settings.get("customFormats", []))
    sync_quality_profiles(api, settings.get("qualityProfiles", []))
    check_quality_profile(api)


def setup_prowlarr(prowlarr, sonarr_key, radarr_key):
    log("Prowlarr:")
    set_login(prowlarr)
    for impl, url, key in (("Sonarr", "http://sonarr:8989", sonarr_key),
                           ("Radarr", "http://radarr:7878", radarr_key)):
        ensure_provider(prowlarr, "applications", impl, impl, {
            "prowlarrUrl": "http://prowlarr:9696",
            "baseUrl": url,
            "apiKey": key,
        }, extra={"syncLevel": "fullSync"})
    if not prowlarr.get("/indexer"):
        log("  note: Prowlarr has no indexers yet. Add them at http://<server>:9696, they sync to Sonarr and Radarr")


def main():
    for name in ("ARR_PASSWORD", "SONARR_API_KEY", "RADARR_API_KEY", "PROWLARR_API_KEY", "WEBHOOK_SECRET"):
        if not os.environ.get(name):
            sys.exit(f"{name} is not set, run the install script to create .env")

    make_media_folders()

    sonarr = Api("Sonarr", "http://sonarr:8989", "v3", os.environ["SONARR_API_KEY"])
    radarr = Api("Radarr", "http://radarr:7878", "v3", os.environ["RADARR_API_KEY"])
    prowlarr = Api("Prowlarr", "http://prowlarr:9696", "v1", os.environ["PROWLARR_API_KEY"])
    for api in (sonarr, radarr, prowlarr):
        api.wait()
    wait_for_url("qBittorrent", f"http://{QBIT['host']}:{QBIT['port']}/")
    wait_for_url("MediaHub", BACKEND_URL + "/health")

    with open(SETTINGS_FILE, encoding="utf-8") as f:
        settings = json.load(f)

    setup_qbit()
    setup_arr(sonarr, settings, "TV Shows", "tvCategory", "tv", "/api/webhooks/sonarr")
    setup_arr(radarr, settings, "Movies", "movieCategory", "movies", "/api/webhooks/radarr")
    setup_prowlarr(prowlarr, sonarr.key, radarr.key)
    log("bootstrap finished")


if __name__ == "__main__":
    if sys.argv[1:] == ["qbit-config"]:
        qbit_config()
    else:
        try:
            main()
        except RuntimeError as e:
            sys.exit(f"bootstrap failed: {e}")
