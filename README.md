# **MediaHub**
## What is this?
This is my attempt at a self hosted media management platform. This handles the curation of a library of media across Movies, Anime, Manga and Music that can then be downloaded to the server for playback through Plex. 

## Included Features
* Media catalogs for Movies, Anime, Manga, and Music
* Request / approval workflow for downloads
* Progress and status tracking per user
* Media playback

## Tools
| Tool | Purpose |
|------|---------|
| **Sonarr** | Handle downloading and library organization of Anime |
| **Radarr** | Handles downloading and library organization of Movies |
| **yt-dlp** | Handles downloading and library organization of Music |
| **mangal** | Handles downloading and library organization of Manga |
| **postgres** | Database |

## Prerequisites
Ensure the following are installed before starting:
- **Node.js** and **npm**
- **Go**
- **PostgreSQL**
- **Goose**
- **Sonarr**
- **Radarr**
- **yt-dlp**
- **mangal**

## Project Structure
```
    MediaHub/
    ├── backend/
    │   ├── cmd/api/        # Application entry point
    │   ├── internal/       # Auth, media, requests, jobs, downloader, etc.
    │   ├── migrations/     # Goose SQL migration files
    │   └── .env
    └── frontend/
        ├── src/
        │   ├── components/ # Shared UI components
        │   ├── hooks/      # React hooks
        │   ├── pages/      # Page components
        │   └── services/   # API service layer
        └── .env
```

## User Roles
| Role | Permissions |
|------|-------------|
| **User** | Browse catalog, submit download requests, track status |
| **Admin** | All user permissions, plus approve/ reject requests, manage jobs and catalog |
Users can be assigned one of two download permissions: `vetted` (requests require admin approval) or `auto-approved` (requests are approved immediately)

## Install (Docker)
One script installs everything MediaHub needs: PostgreSQL, the backend (with ffmpeg, yt-dlp and mangal built in), the web app behind Caddy, qBittorrent, Sonarr, Radarr, Prowlarr, and optionally a Cloudflare Tunnel. It installs Docker if it's missing, generates every secret, and connects the apps to each other. You don't need any UI setup.

**Linux (Ubuntu/Debian)**
```
git clone https://github.com/Mantie7553/MediaHub.git && cd MediaHub
./install.sh
```

**Windows 10/11**
```
git clone https://github.com/Mantie7553/MediaHub.git; cd MediaHub
powershell -ExecutionPolicy Bypass -File .\install.ps1
```
On a PC without Docker, the script installs WSL 2 and Docker Desktop. It then asks you to restart and run it again.

The script asks for:
- **Media folder.** This folder holds `TV Shows`, `Movies`, `Music`, `Manga` and `Light Novels`. You can give it a local folder, or an SMB share (`\nas\media` on Windows; on Linux, answer the SMB question).
- **Address people will use**, e.g. `https://media.example.com`.
- **Cloudflare tunnel token** (optional). Create a named tunnel in Cloudflare Zero Trust, point its public hostname at `http://web:80`, and paste the token.

At the end it prints the logins. The MediaHub admin is `admin@example.com`. The qBittorrent/Sonarr/Radarr/Prowlarr user is `admin`. Both passwords are saved in `.env`. The one manual step left is adding indexers in Prowlarr (`http://<server>:9696`); they sync to Sonarr and Radarr.

What the first-run `bootstrap` container sets up (re-running it changes nothing):
- In qBittorrent: **no seeding**. Torrents stop at ratio 0, and Sonarr/Radarr import them and then remove them. This is re-applied on every run.
- In Sonarr and Radarr:
  - root folders `<media>/TV Shows` and `<media>/Movies`
  - qBittorrent as the download client, with completed and failed downloads removed
  - the MediaHub webhook with `X-MediaHub-Secret`
  - login
  - the custom formats and the **Anime** quality profile from `deploy/bootstrap/arr-settings.json`
- In Prowlarr: Sonarr and Radarr as apps.

**The Anime profile.** MediaHub adds every series and movie with this profile; set `SONARR_QUALITY_PROFILE`/`RADARR_QUALITY_PROFILE` to use a different one. Series are added as the *anime* type, so Sonarr searches by absolute episode number. The profile aims for one file with Japanese audio, English subtitles *and* an English dub, so each user can pick in the player:

| Release | Format score |
|---|---|
| Dual audio (`Dual Audio`, `DUAL`, `JPN+ENG`, `DL`/`ML` tags) | +100 |
| Japanese named as the language | +10 |
| No language named (most fansubs, e.g. SubsPlease, Erai-raws) | 0 |
| English dub only | -5 |
| Raw, no subtitles (`*-Raws`, `RAW`) | -10000, never grabbed |

Qualities are grouped as SD / 720p / 1080p. Within a resolution the score decides, and a higher resolution beats a higher score. Upgrades stay on until the file is 1080p and dual audio. So if only a subbed release exists at first, Sonarr/Radarr swap it for a dual-audio one when one appears.

`arr-settings.json` is the source of truth. Bootstrap brings these formats and the profile back to match it on every install or update run, so change them in the file, not in the Sonarr/Radarr UI. Other profiles and formats are left alone. Per-user audio and subtitle language preferences live in MediaHub's database and move with it (see the migration steps below).

## Updating
- **Automatic.** Every push to `main` runs `.github/workflows/release.yml`. It tests the backend, builds the frontend, then publishes the `mediahub-backend`, `mediahub-web` and `mediahub-bootstrap` images to GHCR. On each install, the `watchtower` container checks every 5 minutes, pulls new MediaHub images and restarts them. Database migrations run automatically when the backend starts.
- **Manual** (`./update.sh` or `.\update.ps1`). This runs `git pull`, pulls all images (including new Sonarr/Radarr/qBittorrent versions), applies changes to `docker-compose.yml` or `.env`, and re-runs bootstrap. Use it after changing `.env` or the compose files.
- **Roll back.** Set `MEDIAHUB_TAG=sha-<commit>` in `.env` and run `docker compose up -d`. Set it back to `latest` to follow `main` again.
- **One-time setup on GitHub.** After the first workflow run, open each of the 3 packages under *Your profile → Packages → Package settings* and set visibility to **Public**. Until then, installs can't pull them (GHCR packages start private) and fall back to building locally.

## Migrating an existing (non-Docker) install
1. Back up the old database: `pg_dump -Fc -U <user> <db> > mediahub.dump`.
2. Stop the old services so their ports are free. For example: `sudo systemctl disable --now sonarr radarr prowlarr qbittorrent-nox caddy cloudflared postgresql` and the MediaHub backend.
3. In `.env`, set `MEDIA_HOST_PATH` and `MEDIA_ROOT` to the **same** path the old install used (e.g. `/mnt/nas/Media`). File paths stored in the database then still match. `install.sh` does this by default.
4. Start only the database and restore into it:
   ```
   docker compose up -d db
   docker compose exec -T db pg_restore -U mediahub -d mediahub --clean --if-exists --no-owner < mediahub.dump
   ```
   Then run `./install.sh` again to start the rest.
5. Sonarr/Radarr start fresh, and bootstrap wires them up. To keep their libraries, add your series and movies again from the existing folders (*Library Import*), or copy the old config folders into the `sonarr-config`/`radarr-config` volumes before the first start.

## Setup (development, without Docker)
*Setup steps for front and back end servers + database*
### *Database*
1. Create a PostgreSQL database
2. Migrations run automatically when the backend starts. To run them by hand from `MediaHub/backend` with Goose:  
```goose -dir ./migrations postgres "host=<your_host> port=<port_number> user=<username> password=<password> dbname=<database_name> sslmode=disable" up```
### *Back End*
1. Create a `.env` file for the environment variables in `/MediaHub/backend`:
    | Variable | Description |
    |----------|-------------|
    | `SONARR_URL` | URL for making requests to Sonarr |
    | `SONARR_API_KEY` | API key for your Sonarr instance |
    | `RADARR_URL` | URL for making requests to Radarr | 
    | `RADARR_API_KEY` | API key for your Radarr instance |
    | `JWT_SECRET` | Secret key for signing JWTs |
    | `MEDIA_SIGNING_SECRET` | Secret key for signing short lived media URLs. Must be different from `JWT_SECRET` |
    | `WEBHOOK_SECRET` | Shared secret Sonarr/Radarr send in the `X-MediaHub-Secret` header |
    | `DB_HOST` | Database hostname |
    | `DB_PORT` | Database port |
    | `DB_USER` | Database username |
    | `DB_PASSWORD` | Database password |
    | `DB_NAME` | Database name |
    | `MEDIA_ROOT` | Root path where media will be downloaded |
    | `MANGAL_PATH` | Path to the mangal executable |
    | `TRANSCODE_HWACCEL` | *(Optional)* GPU decoder for AV1 videos. Defaults to `d3d11va` on Windows; on Linux set e.g. `vaapi`, or `none` to decode on the CPU |
2. In both Sonarr and Radarr, go to *Settings → Connect → + → Webhook*:
    - **URL:** `<api_url>/api/webhooks/sonarr` (or `/radarr`)
    - **Method:** `POST`
    - **Headers:** add `X-MediaHub-Secret` with the value of `WEBHOOK_SECRET`

    Webhook calls without the header are rejected. Don't put the secret in the URL, it would end up in access logs.
3. Sign up is invite only. Log in as an admin, go to *User Management → Invites → Create Invite*, and share the code or link it gives you. Each code works for one sign up.
4. *(Optional)* For local testing, create `/MediaHub/backend/.env.local`. Any variable in it overrides `.env`, and it is gitignored. For example, to use local storage with the same folder layout instead of the NAS:
    ```
    MEDIA_ROOT=S:/Media
    ```
    Delete or rename the file to go back to the values in `.env`. Use a trailing slash for a bare drive (`Z:/`, not `Z:`), otherwise Windows treats paths as relative to that drive's current folder.
5. Start the server from **"/MediaHub/backend"**:  
 ```go run .\cmd\api\main.go```

### *Front End*
1. Create a `.env` file in `/MediaHub/frontend`:
    | Variable | Description |
    |----------|-------------|
    | `VITE_API_URL` | URL the API server can be reached at |
2. Install dependencies and start the server from `/MediaHub/frontend`:  
`npm i`  
`npm run dev`