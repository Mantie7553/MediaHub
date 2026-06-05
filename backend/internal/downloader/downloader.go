package downloader

import (
	"archive/zip"
	"bufio"
	"bytes"
	"database/sql"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
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

	logger.Info("Job %s completed!", jobID)
	db.Exec(`UPDATE download_jobs 
        SET status = 'complete', progress_pct = 100, completed_at = NOW()
        WHERE id = $1`, jobID)
}

func RunMangal(db *sql.DB, jobID string, mediaItemID string, sourceURL string, dest string) {
	logger.Info("Mangal running job %s", jobID)
	row := db.QueryRow(`
        UPDATE download_jobs 
        SET status = 'downloading', started_at = NOW()
        WHERE id = $1
        RETURNING source_url, destination_path`,
		jobID,
	)

	if row.Err() != nil {
		markFailed(db, jobID, row.Err().Error())
		return
	}

	var mangaTitle string
	err := db.QueryRow(`SELECT title FROM media_items WHERE id = $1`, mediaItemID).Scan(&mangaTitle)

	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	mangalPath := os.Getenv("MANGAL_PATH")
	if mangalPath == "" {
		mangalPath = "mangal"
	}

	cmd := exec.Command(mangalPath, "inline",
		"-q", mangaTitle,
		"-m", "0",
		"-c", "all",
		"-d",
		"-S", "Mangadex",
	)

	cmd.Env = append(os.Environ(),
		"MANGAL_DOWNLOADER_PATH="+dest,
		"MANGAL_FORMATS_USE=cbz",
	)

	_, err = cmd.CombinedOutput()
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	// find the most recently modified directory
	entries, _ := os.ReadDir(dest)
	var mangaDir string
	var newestTime int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > newestTime {
			newestTime = info.ModTime().UnixNano()
			mangaDir = filepath.Join(dest, e.Name())
		}
	}

	if mangaDir == "" {
		markFailed(db, jobID, "could not find downloaded manga directory")
		return
	}

	err = filepath.WalkDir(mangaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Error("WalkDir failed at %s: %s", path, err.Error())
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".cbz" {
			return nil
		}

		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		chapterNum := 0.0
		if strings.HasPrefix(base, "[") {
			end := strings.Index(base, "]")
			if end > 1 {
				chapterNum, _ = strconv.ParseFloat(strings.TrimSpace(base[1:end]), 64)
			}
		}

		r, err := zip.OpenReader(path)
		pageCount := 0
		if err == nil {
			for _, f := range r.File {
				ext := strings.ToLower(filepath.Ext(f.Name))
				if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
					pageCount++
				}
			}
			r.Close()
		}

		db.Exec(
			`INSERT INTO manga_chapters (media_item_id, chapter_number, file_path, page_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING`,
			mediaItemID, chapterNum, path, pageCount,
		)
		return nil
	})

	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	db.Exec(`UPDATE download_jobs 
		SET status = 'complete', progress_pct = 100, completed_at = NOW()
		WHERE id = $1`, jobID)
	logger.Info("Mangal completed job %s", jobID)
}

func markFailed(db *sql.DB, jobID string, message string) {
	logger.Error("JOB-%s failed: %s", jobID, message)
	db.Exec(`UPDATE download_jobs 
        SET status = 'failed', error_message = $1, completed_at = NOW()
        WHERE id = $2`, message, jobID)
}
