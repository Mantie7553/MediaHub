import { useState, useEffect, useRef } from "react";
import api from "../../services/api";
import Format from "../../utils/format";
import { errorMessage } from "../../utils/errors";
import { BookOpen, BookText, Clapperboard, Music, RefreshCw, Tv } from "lucide-react";
import { Panel } from "../../components/DataTable";
import StatusPill from "../../components/StatusPill";

const SOURCE_LABELS = {
    "sonarr": "TV (Sonarr)",
    "radarr": "Movies (Radarr)",
    "manga": "Manga",
    "light-novels": "Light Novels",
    "music": "Music",
};

const SOURCE_ICONS = {
    "sonarr": Tv,
    "radarr": Clapperboard,
    "manga": BookOpen,
    "light-novels": BookText,
    "music": Music,
};

const POLL_MS = 3000;

/**
 * Admin section that runs a full library sync (Sonarr, Radarr, manga, light novels, music)
 * and shows the progress and results of the current or last run
 */
export default function LibrarySyncSection() {
    const [status, setStatus] = useState(null);
    const [error, setError] = useState("");
    const pollRef = useRef(null);

    function fetchStatus() {
        return api.get("/admin/sync")
            .then(res => {
                setStatus(res.data);
                return res.data;
            })
            .catch(err => setError(errorMessage(err, "Could not load the sync status")));
    }

    // poll while a sync is running, stop once it's done
    function poll() {
        clearTimeout(pollRef.current);
        fetchStatus().then(data => {
            if (data?.running) pollRef.current = setTimeout(poll, POLL_MS);
        });
    }

    useEffect(() => {
        poll();
        return () => clearTimeout(pollRef.current);
    }, []);

    function handleStart() {
        setError("");
        api.post("/admin/sync")
            .then(res => {
                setStatus(res.data);
                pollRef.current = setTimeout(poll, POLL_MS);
            })
            .catch(err => {
                // 409: already running, just show its progress
                if (err.response?.status === 409) poll();
                else setError(errorMessage(err, "Could not start the sync"));
            });
    }

    const running = status?.running;
    const results = status?.results ?? [];
    const done = new Set(results.map(r => r.source));

    return (
        <Panel
            title="Library Sync"
            icon={RefreshCw}
            description={<>
                Adds anything new from Sonarr, Radarr and the Manga, Light Novels and Music folders.
                {status?.next_run
                    ? ` Runs every night, next on ${Format.dateTime(status.next_run)}.`
                    : " The nightly sync is turned off."}
            </>}
            actions={<button className="btn btn-sm btn-primary" onClick={handleStart} disabled={!status || running}>
                {running ? <span className="loading loading-spinner loading-xs" /> : <RefreshCw size={15}/>}
                {running ? "Syncing…" : "Sync now"}
            </button>}
        >
            {error && <p role="alert" className="text-error text-sm mb-3">{error}</p>}

            {status?.started_at && <>
                <ul className="flex flex-col gap-1">
                    {Object.keys(SOURCE_LABELS).map(source => {
                        const result = results.find(r => r.source === source);
                        const Icon = SOURCE_ICONS[source];
                        return (
                            <li key={source} className="flex items-center gap-3 rounded-field px-3 py-2.5 bg-white/[0.03] ring-1 ring-white/5">
                                <Icon size={17} className="shrink-0 text-base-content/45"/>
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium">{SOURCE_LABELS[source]}</p>
                                    {result && !result.ok && <p className="text-xs text-error break-words">{result.error}</p>}
                                </div>
                                {result?.duration && <span className="hidden sm:inline text-xs tabular-nums text-base-content/45">{result.duration}</span>}
                                {result ? (
                                    result.ok
                                        ? <StatusPill label={result.detail || "done"} tone="success"/>
                                        : <StatusPill label="failed" tone="error"/>
                                ) : running && !done.has(source) ? (
                                    <StatusPill label="waiting" tone="info" pulse/>
                                ) : (
                                    <StatusPill label="not run"/>
                                )}
                            </li>
                        );
                    })}
                </ul>
                <p className="text-xs text-base-content/45 mt-3">
                    {running ? "Started" : "Last run"} {Format.dateTime(status.started_at)}
                    {status.trigger === "schedule" ? " (nightly)" : " (manual)"}
                </p>
            </>}
        </Panel>
    );
}
