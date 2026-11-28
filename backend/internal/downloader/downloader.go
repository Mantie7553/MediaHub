package downloader

import (
	"bufio"
	"bytes"
	"database/sql"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/notify"
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
	notify.JobDownloading(db, jobID)

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

	// age-restricted videos need a signed-in YouTube account's cookies
	cookiesFile, err := cookiesCopy()
	if err != nil {
		logger.Warn("Job %s can't use the yt-dlp cookies file: %s", jobID, err.Error())
	}
	if cookiesFile != "" {
		defer os.Remove(cookiesFile)
	}

	// 4. Build and start the command
	ytdlpPath := os.Getenv("YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}
	cmd := exec.Command(ytdlpPath, ytdlpArgs(outputTemplate, tmpFile.Name(), cookiesFile, sourceURL)...)
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
		markFailed(db, jobID, failureMessage(stderr.String(), cookiesFile != ""))
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
	notify.JobArrived(db, jobID, 1)
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
Function: cookiesCopy
Purpose: a private copy of the YouTube cookies file YTDLP_COOKIES names, for one download.
yt-dlp rewrites its cookies file when it finishes, so it gets a copy: the original can be mounted
read-only and two downloads never write the same file. The copy goes in YTDLP_TMPDIR, on the
container's own disk, not the media disk. Returns "" when no cookies file is set up; the caller
removes the copy
*/
func cookiesCopy() (string, error) {
	src := os.Getenv("YTDLP_COOKIES")
	if src == "" {
		return "", nil
	}
	in, err := os.Open(src)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.CreateTemp(os.Getenv("YTDLP_TMPDIR"), "ytdlp-cookies-*.txt")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(out.Name())
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(out.Name())
		return "", err
	}
	return out.Name(), nil
}

/*
Function: failureMessage
Purpose: what a failed download's job shows. An age-restricted video says how to fix it; the
full yt-dlp output follows either way
Params:
  - stderr: yt-dlp's error output
  - hadCookies: whether the download used a cookies file
*/
func failureMessage(stderr string, hadCookies bool) string {
	if !strings.Contains(stderr, "confirm your age") {
		return stderr
	}
	hint := "Age-restricted on YouTube: put a signed-in account's cookies.txt in the yt-dlp folder (see .env.example)"
	if hadCookies {
		hint = "Age-restricted on YouTube and the cookies were refused, they have likely expired: export a fresh cookies.txt (see .env.example)"
	}
	return hint + "\n\n" + stderr
}

/*
Function: ytdlpArgs
Purpose: the yt-dlp arguments for downloading one track as mp3 with its thumbnail as album art.
The source comes last, after --, so yt-dlp reads it as a URL even if it starts with a dash
Params:
  - outputTemplate: where to save the file, with yt-dlp's %(title)s and %(ext)s fields
  - printFile: a file yt-dlp writes the finished file's path to
  - cookiesFile: a YouTube cookies file to sign in with, "" for none
  - sourceURL: the canonical YouTube link
*/
func ytdlpArgs(outputTemplate, printFile, cookiesFile, sourceURL string) []string {
	args := []string{
		"-x",
		"--audio-format", "mp3",
		// the video's thumbnail becomes the track's album art
		"--embed-thumbnail",
		"--convert-thumbnails", "jpg",
		"--newline",
		"-o", outputTemplate,
		"--print-to-file", "after_move:filepath", printFile,
	}
	if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}
	// everything after -- is a URL, never an option, whatever it starts with
	return append(args, "--", sourceURL)
}

/*
Function: markFailed
Purpose: mark a job failed with why, and tell the admins
*/
func markFailed(db *sql.DB, jobID string, message string) {
	logger.Error("JOB-%s failed: %s", jobID, message)
	db.Exec(`UPDATE download_jobs 
        SET status = 'failed', error_message = $1, completed_at = NOW()
        WHERE id = $2`, message, jobID)
	notify.JobFailed(db, jobID, message)
}
