package downloader

import (
	"bufio"
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
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

	// the link was checked when the job was made; checking again here covers jobs saved
	// before that check existed
	sourceURL, err = ytdlp.CanonicalURL(sourceURL)
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	// 3. Build output path - fall back to flat structure if no metadata.
	// The names become folders: cleaned so they can't climb out of the library or carry a
	// yt-dlp template field
	artist, album = musicfiles.CleanName(artist), musicfiles.CleanName(album)
	if album == "" {
		album = "Singles"
	}
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
	cmd := exec.Command(ytdlpPath, ytdlpArgs(outputTemplate, tmpFile.Name(), sourceURL)...)
	cmd.Env = ytdlp.Env()

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
			saveCover(db, mediaItemID, filePath)
		}
	}

	logger.Info("Job %s completed!", jobID)
	db.Exec(`UPDATE download_jobs 
        SET status = 'complete', progress_pct = 100, completed_at = NOW()
        WHERE id = $1`, jobID)
}

/*
Function: saveCover
Purpose: give a downloaded track's album folder a cover from the track's embedded art when it
has none yet, and make it the album's cover if the album has none
Params:
  - db: a pointer to our database
  - mediaItemID: the track's media item id
  - filePath: the downloaded track
*/
func saveCover(db *sql.DB, mediaItemID, filePath string) {
	coverPath := musicfiles.CoverPath(filePath)
	if _, err := os.Stat(coverPath); err != nil {
		if !os.IsNotExist(err) || !musicfiles.ExtractCover(filePath, coverPath) {
			return
		}
	}
	db.Exec(`
        UPDATE albums SET cover_image_url = $1
        WHERE id = (SELECT album_id FROM music_metadata WHERE media_item_id = $2)
        AND cover_image_url IS NULL`,
		musicfiles.CoverURL(mediaItemID), mediaItemID)
}

/*
Function: ytdlpArgs
Purpose: the yt-dlp arguments for downloading one track as mp3 with its thumbnail as album art.
The source comes last, after --, so yt-dlp reads it as a URL even if it starts with a dash
Params:
  - outputTemplate: where to save the file, with yt-dlp's %(title)s and %(ext)s fields
  - printFile: a file yt-dlp writes the finished file's path to
  - sourceURL: the canonical YouTube link
*/
func ytdlpArgs(outputTemplate, printFile, sourceURL string) []string {
	return []string{
		"-x",
		"--audio-format", "mp3",
		// the video's thumbnail becomes the track's album art
		"--embed-thumbnail",
		"--convert-thumbnails", "jpg",
		"--newline",
		"-o", outputTemplate,
		"--print-to-file", "after_move:filepath", printFile,
		// everything after -- is a URL, never an option, whatever it starts with
		"--", sourceURL,
	}
}

func markFailed(db *sql.DB, jobID string, message string) {
	logger.Error("JOB-%s failed: %s", jobID, message)
	db.Exec(`UPDATE download_jobs 
        SET status = 'failed', error_message = $1, completed_at = NOW()
        WHERE id = $2`, message, jobID)
}
