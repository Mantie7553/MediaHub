package downloader

import (
	"bufio"
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
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
	logger.Info("Starting job %s", jobID)
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
		logger.Warn("Job %s missing artist information", jobID)
		outputTemplate = destinationPath + "/%(title)s.%(ext)s"
	}

	tmpFile, err := os.CreateTemp("", "ytdlp-path-*.txt")
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// 4. Build and start the command
	ytdlpPath := os.Getenv("YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}
	cmd := exec.Command(ytdlpPath,
		"-x",
		"--audio-format", "mp3",
		"--newline",
		"-o", outputTemplate,
		sourceURL,
		"--print-to-file", "after_move:filepath", tmpFile.Name(),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

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
		markFailed(db, jobID, stderr.String())
		return
	}

	// 7. Save the downloaded file path
	pathBytes, err := os.ReadFile(tmpFile.Name())
	if err == nil {
		filePath := strings.TrimSpace(string(pathBytes))
		if filePath != "" {
			db.Exec(`UPDATE music_metadata SET file_path = $1 WHERE media_item_id = $2`,
				filePath, mediaItemID)
		}
	}

	logger.Info("Job %s completed!", jobID)
	db.Exec(`UPDATE download_jobs 
        SET status = 'complete', progress_pct = 100, completed_at = NOW()
        WHERE id = $1`, jobID)
}

func markFailed(db *sql.DB, jobID string, message string) {
	logger.Error("JOB-%s failed: %s", jobID, message)
	db.Exec(`UPDATE download_jobs 
        SET status = 'failed', error_message = $1, completed_at = NOW()
        WHERE id = $2`, message, jobID)
}
