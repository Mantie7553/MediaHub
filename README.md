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

## Setup
*Setup steps for front and back end servers + database*
### *Database*
1. Create a PostgreSQL database
2. Run migrations from `MediaHub/backend` using Goose:  
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
    | `DB_HOST` | Database hostname |
    | `DB_PORT` | Database port |
    | `DB_USER` | Database username |
    | `DB_PASSWORD` | Database password |
    | `DB_NAME` | Database name |
    | `MEDIA_ROOT` | Root path where media will be downloaded |
    | `MANGAL_PATH` | Path to the mangal executable |
2. Start the server from **"/MediaHub/backend"**:  
 ```go run .\cmd\api\main.go```

### *Front End*
1. Create a `.env` file in `/MediaHub/frontend`:
    | Variable | Description |
    |----------|-------------|
    | `VITE_API_URL` | URL the API server can be reached at |
2. Install dependencies and start the server from `/MediaHub/frontend`:  
`npm i`  
`npm run dev`