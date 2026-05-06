package downloader

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/arr"
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
		handler = "mangal"
	default:
		dest = filepath.Join(mediaRoot, "Downloads")
		handler = "ytdlp"
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

	// handle each of the different download clients
	switch handler {
	case "sonarr":
		// needs external ID
		if externalID == nil {
			return "", fmt.Errorf("media item has no external_id")
		}

		extIDInt, err := strconv.Atoi(*externalID)
		if err != nil {
			return "", fmt.Errorf("invalid external_id")
		}
		// connect to sonarr and start downloading
		sClient := arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")
		seriesID, err := sClient.AddSeries(extIDInt, 1, dest)

		if err != nil {
			return "", fmt.Errorf("internal server error")
		}

		// save the new content as a sonarr item in the database
		db.Exec(
			`INSERT INTO sonarr_items (media_item_id, sonarr_series_id) 
			VALUES ($1, $2)`,
			mediaItemID, seriesID,
		)

	case "radarr":
		// needs external ID
		if externalID == nil {
			return "", fmt.Errorf("media item has no external_id")
		}

		extIDInt, err := strconv.Atoi(*externalID)
		if err != nil {
			return "", fmt.Errorf("invalid external_id")
		}
		// connect to radarr and start downloading
		rClient := arr.NewArrClient("RADARR_URL", "RADARR_API_KEY")
		movieID, err := rClient.AddMovie(extIDInt, 1, dest)

		if err != nil {
			return "", fmt.Errorf("internal server error")
		}

		// save the content as a radarr item in the database
		db.Exec(
			`INSERT INTO radarr_items (media_item_id, radarr_movie_id) 
			VALUES ($1, $2)`,
			mediaItemID, movieID,
		)

	case "mangal":
		go RunMangal(db, jobID, mediaItemID, sourceURL, dest)

	// if it is not any of the above try downloaing it with ytdlp
	default:
		go Run(db, jobID, mediaItemID)
	}

	// return the new id and no error
	return jobID, nil
}
