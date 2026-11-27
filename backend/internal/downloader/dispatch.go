package downloader

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/notify"
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
	// an admin downloads these themselves: nothing to start
	if ByHand(mediaType) {
		logger.Info("Request %s of type %s is downloaded by hand, no job started", requestID, mediaType)
		return "", nil
	}

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

	// Sonarr and Radarr couldn't take it: the job is failed with why, so it isn't left queued
	// forever, and the admins are told
	fail := func(reason string, err error) (string, error) {
		markFailed(db, jobID, reason)
		return "", err
	}

	// handle each of the different download clients
	switch handler {
	case "sonarr":
		// needs external ID
		if externalID == nil {
			return fail("the title has no external id to find it in Sonarr", fmt.Errorf("media item has no external_id"))
		}

		// connect to sonarr and start downloading
		sClient := arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")

		var title string
		var knownTvdb sql.NullInt64
		err = db.QueryRow(`SELECT title, tvdb_id FROM media_items WHERE id = $1`, mediaItemID).Scan(&title, &knownTvdb)
		if err != nil {
			return fail(err.Error(), fmt.Errorf("failed to get title: %w", err))
		}

		// the series by its TVDB id when known, so a season's own name ("Nisemonogatari") can't
		// pick the wrong series
		term := title
		if knownTvdb.Valid {
			term = fmt.Sprintf("tvdb:%d", knownTvdb.Int64)
		}
		tvdbID, sonarrID, err := sClient.LookupSeries(term)
		if err != nil {
			return fail("Sonarr lookup failed: "+err.Error(), fmt.Errorf("sonarr lookup failed: %w", err))
		}

		var seriesID int
		if sonarrID > 0 {
			seriesID = sonarrID
		} else {
			seriesID, err = sClient.AddSeries(tvdbID, qualityProfile(sClient, "SONARR_QUALITY_PROFILE"), dest, title)
			if err != nil {
				return fail("Sonarr: "+err.Error(), fmt.Errorf("internal server error"))
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
		notify.JobDownloading(db, jobID)
		if _, err := db.Exec(
			`UPDATE media_items SET tvdb_id = $1 WHERE id = $2 AND tvdb_id IS NULL
			AND NOT EXISTS (SELECT 1 FROM media_items o WHERE o.type = 'anime' AND o.tvdb_id = $1)`,
			tvdbID, mediaItemID,
		); err != nil {
			logger.Error("failed to save the TVDB id of %s: %s", title, err.Error())
		}

	case "radarr":
		// needs external ID
		if externalID == nil {
			return fail("the title has no external id to find it in Radarr", fmt.Errorf("media item has no external_id"))
		}

		extIDInt, err := strconv.Atoi(*externalID)
		if err != nil {
			return fail("the title's external id isn't a TMDB id", fmt.Errorf("invalid external_id"))
		}
		// connect to radarr and start downloading
		rClient := arr.NewArrClient("RADARR_URL", "RADARR_API_KEY")
		movieID, err := rClient.AddMovie(extIDInt, qualityProfile(rClient, "RADARR_QUALITY_PROFILE"), dest)

		if err != nil {
			return fail("Radarr: "+err.Error(), fmt.Errorf("internal server error"))
		}

		// save the content as a radarr item in the database
		if _, err := db.Exec(
			`INSERT INTO radarr_items (media_item_id, radarr_movie_id) 
			VALUES ($1, $2)`,
			mediaItemID, movieID,
		); err != nil {
			logger.Error("failed to save radarr_id %s", err.Error())
		}
		notify.JobDownloading(db, jobID)

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
	rows, err := db.Query(
		`UPDATE download_jobs
		SET status = 'failed', error_message = $1, completed_at = NOW()
		WHERE status IN ('queued', 'downloading') AND handler IN ('ytdlp', 'mangadex')
		RETURNING id`,
		interruptedMessage,
	)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	// the admins are told, after the rows are read so the lookups don't wait on them
	for _, id := range ids {
		notify.JobFailed(db, id, interruptedMessage)
	}
	return int64(len(ids)), nil
}
