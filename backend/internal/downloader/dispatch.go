package downloader

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

/*
Function:	Dispatch
Purpose:	Handle where and how media will be downloaded
Params:
  - db: a connection to the database
  - requestID: the id from the incoming request
  - mediaItemID: the id of the media to be downloaded
  - sourceURL: the url the media is coming from
  - mediaType: the type of media to download
  - externalID: the id from the external API/database (optional)
*/
func Dispatch(db *sql.DB, requestID string, mediaItemID string,
	sourceURL string, mediaType string, externalID *string) (string, error) {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	var dest string
	var handler string

	// set where to install media to and what will handle the downloading
	switch mediaType {
	case "anime":
		dest = filepath.Join(mediaRoot, "TV Shows")
		handler = "sonarr"
	case "movie":
		dest = filepath.Join(mediaRoot, "Movies")
		handler = "radarr"
	case "music_track":
		dest = filepath.Join(mediaRoot, "Music")
		handler = "ytdlp"
	case "manga":
		dest = filepath.Join(mediaRoot, "Manga")
		handler = "mangadex"
	default:
		dest = filepath.Join(mediaRoot, "Downloads")
		handler = "ytdlp"
	}

	// yt-dlp only ever gets a link rebuilt from a YouTube video id: whatever reached here (a
	// request saved before this check, an admin creating a job) can't pass it options or
	// point it at another site
	if handler == "ytdlp" {
		canonical, err := ytdlp.CanonicalURL(sourceURL)
		if err != nil {
			return "", err
		}
		sourceURL = canonical
	}

	// add a new download job to the database
	var jobID string
	err := db.QueryRow(
		`INSERT INTO download_jobs (request_id, media_item_id, source_url, destination_path, handler)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		requestID, mediaItemID, sourceURL, dest, handler,
	).Scan(&jobID)

	if err != nil {
		return "", fmt.Errorf("insert job failed: %w", err)
	}
	logger.Info("Dispatched %s of type %s. Handled by %s", requestID, mediaType, handler)

	// handle each of the different download clients
	switch handler {
	case "sonarr":
		// needs external ID
		if externalID == nil {
			logger.Error("External ID is null")
			return "", fmt.Errorf("media item has no external_id")
		}

		// connect to sonarr and start downloading
		sClient := arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")

		var title string
		err = db.QueryRow(`SELECT title FROM media_items WHERE id = $1`, mediaItemID).Scan(&title)
		if err != nil {
			return "", fmt.Errorf("failed to get title: %w", err)
		}

		tvdbID, sonarrID, err := sClient.LookupSeries(title)
		if err != nil {
			logger.Error("Sonarr lookup failed: %s", err.Error())
			return "", fmt.Errorf("sonarr lookup failed: %w", err)
		}

		var seriesID int
		if sonarrID > 0 {
			seriesID = sonarrID
		} else {
			seriesID, err = sClient.AddSeries(tvdbID, qualityProfile(sClient, "SONARR_QUALITY_PROFILE"), dest, title)
			if err != nil {
				logger.Error("Sonarr: %s", err.Error())
				return "", fmt.Errorf("internal server error")
			}
		}

		// save the new content as a sonarr item in the database
		if _, err := db.Exec(
			`INSERT INTO sonarr_items (media_item_id, sonarr_series_id) 
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO NOTHING`,
			mediaItemID, seriesID,
		); err != nil {
			logger.Error("failed to save sonarr_id %s", err.Error())
		}

	case "radarr":
		// needs external ID
		if externalID == nil {
			logger.Error("External ID is null")
			return "", fmt.Errorf("media item has no external_id")
		}

		extIDInt, err := strconv.Atoi(*externalID)
		if err != nil {
			logger.Error("External ID is not a number")
			return "", fmt.Errorf("invalid external_id")
		}
		// connect to radarr and start downloading
		rClient := arr.NewArrClient("RADARR_URL", "RADARR_API_KEY")
		movieID, err := rClient.AddMovie(extIDInt, qualityProfile(rClient, "RADARR_QUALITY_PROFILE"), dest)

		if err != nil {
			logger.Error("Radarr: %s", err.Error())
			return "", fmt.Errorf("internal server error")
		}

		// save the content as a radarr item in the database
		if _, err := db.Exec(
			`INSERT INTO radarr_items (media_item_id, radarr_movie_id) 
			VALUES ($1, $2)`,
			mediaItemID, movieID,
		); err != nil {
			logger.Error("failed to save radarr_id %s", err.Error())
		}

	case "mangadex":
		go RunMangaDex(db, jobID, mediaItemID, dest)

	// if it is not any of the above try downloading it with ytdlp
	default:
		go Run(db, jobID, mediaItemID)
	}

	// return the new id and no error
	return jobID, nil
}

/*
Function:	qualityProfile
Purpose:	Quality profile id for new series / movies: the profile named in envKey, "Anime" by
default (created by the Docker bootstrap). Falls back to profile 1 if it doesn't exist
Params:
  - client: the Sonarr or Radarr client
  - envKey: environment variable holding the profile name
*/
func qualityProfile(client *arr.ArrClient, envKey string) int {
	name := os.Getenv(envKey)
	if name == "" {
		name = "Anime"
	}
	id, err := client.QualityProfileID(name)
	if err != nil {
		logger.Warn("quality profile %q not usable (%s), using profile 1", name, err.Error())
		return 1
	}
	return id
}

// the message on a job the server was running when it stopped
const interruptedMessage = "interrupted by a server restart, request it again to finish"

/*
Function:	FailInterrupted
Purpose:	at startup, mark yt-dlp and MangaDex jobs still queued or downloading as failed.
They run inside this server, so nothing can still be working on them, and a job left
"downloading" would block deleting its title forever. Sonarr and Radarr jobs carry on in
their own apps and are left alone
Params:
  - db: a pointer to our database
*/
func FailInterrupted(db *sql.DB) (int64, error) {
	res, err := db.Exec(
		`UPDATE download_jobs
		SET status = 'failed', error_message = $1, completed_at = NOW()
		WHERE status IN ('queued', 'downloading') AND handler IN ('ytdlp', 'mangadex')`,
		interruptedMessage,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
