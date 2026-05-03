package downloader

import (
	"bufio"
	"database/sql"
	"os/exec"
	"strconv"
	"strings"
)

/*
Function: Run
Purpose: Run the download command for music
Params:
  - db: a pointer to our database
  - jobID: an id used to tell the jobs apart
*/
func Run(db *sql.DB, jobID string, mediaItemID string) {
	// 1. Mark as downloading and get source/destination
	var sourceURL, destinationPath string
	err := db.QueryRow(`
        UPDATE download_jobs 
        SET status = 'downloading', started_at = NOW()
        WHERE id = $1
        RETURNING source_url, destination_path`,
		jobID,
	).Scan(&sourceURL, &destinationPath)
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	// 2. Get artist and album from music_metadata
	var artist, album string
	db.QueryRow(`
        SELECT mm.artist, COALESCE(a.title, 'Singles')
        FROM music_metadata mm
        LEFT JOIN albums a ON a.id = mm.album_id
        WHERE mm.media_item_id = $1`,
		mediaItemID,
	).Scan(&artist, &album)

	// 3. Build output path - fall back to flat structure if no metadata
	var outputTemplate string
	if artist != "" {
		outputTemplate = destinationPath + "/" + artist + "/" + album + "/%(title)s.%(ext)s"
	} else {
		outputTemplate = destinationPath + "/%(title)s.%(ext)s"
	}

	// 4. Build and start the command
	cmd := exec.Command("yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"--newline",
		"-o", outputTemplate,
		sourceURL,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	if err := cmd.Start(); err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	// 5. Read progress
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "[download]") && strings.Contains(line, "%") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				percentStr := strings.TrimSuffix(fields[1], "%")
				pct, err := strconv.ParseFloat(percentStr, 64)
				if err == nil {
					db.Exec(`UPDATE download_jobs SET progress_pct = $1 WHERE id = $2`,
						int(pct), jobID)
				}
			}
		}
	}

	// 6. Wait for finish
	if err := cmd.Wait(); err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	db.Exec(`UPDATE download_jobs 
        SET status = 'complete', progress_pct = 100, completed_at = NOW()
        WHERE id = $1`, jobID)
}

func markFailed(db *sql.DB, jobID string, message string) {
	db.Exec(`UPDATE download_jobs 
        SET status = 'failed', error_message = $1, completed_at = NOW()
        WHERE id = $2`, message, jobID)
}
