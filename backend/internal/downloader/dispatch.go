package downloader

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/arr"
)

func Dispatch(db *sql.DB, requestID string, mediaItemID string,
	sourceURL string, mediaType string, externalID *string) (string, error) {
	var dest string
	var handler string

	switch mediaType {
	case "anime":
		dest = "/Media/TV Shows/"
		handler = "sonarr"
	case "movie":
		dest = "/Media/Movies/"
		handler = "radarr"
	case "music_track":
		dest = "/Media/Music/"
		handler = "ytdlp"
	default:
		dest = "/Media/Downloads/"
		handler = "ytdlp"
	}

	var jobID string
	err := db.QueryRow(
		`INSERT INTO download_jobs (request_id, media_item_id, source_url, destination_path, handler)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		requestID, mediaItemID, sourceURL, dest, handler,
	).Scan(&jobID)

	if err != nil {
		return "", fmt.Errorf("internal server error")
	}

	switch handler {
	case "sonarr":
		if externalID == nil {
			return "", fmt.Errorf("media item has no external_id")
		}

		extIDInt, err := strconv.Atoi(*externalID)
		if err != nil {
			return "", fmt.Errorf("invalid external_id")
		}
		sClient := arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")
		seriesID, err := sClient.AddSeries(extIDInt, 1, dest)

		if err != nil {
			return "", fmt.Errorf("internal server error")
		}

		db.Exec(
			`INSERT INTO sonarr_items (media_item_id, sonarr_series_id) 
			VALUES ($1, $2)`,
			mediaItemID, seriesID,
		)

	case "radarr":
		if externalID == nil {
			return "", fmt.Errorf("media item has no external_id")
		}

		extIDInt, err := strconv.Atoi(*externalID)
		if err != nil {
			return "", fmt.Errorf("invalid external_id")
		}
		rClient := arr.NewArrClient("RADARR_URL", "RADARR_API_KEY")
		movieID, err := rClient.AddMovie(extIDInt, 1, dest)

		if err != nil {
			return "", fmt.Errorf("internal server error")
		}

		db.Exec(
			`INSERT INTO radarr_items (media_item_id, radarr_movie_id) 
			VALUES ($1, $2)`,
			mediaItemID, movieID,
		)

	default:
		go Run(db, jobID)
	}

	return jobID, nil
}
