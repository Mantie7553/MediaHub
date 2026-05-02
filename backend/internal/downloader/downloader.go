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
func Run(db *sql.DB, jobID string) {
    // 1. Mark as downloading
    var sourceURL, destinationPath string
    err := db.QueryRow(`
        UPDATE download_jobs 
        SET status = 'downloading', started_at = NOW()
        WHERE id = $1
        RETURNING source_url, destination_path`,
        jobID,
    ).Scan(&sourceURL, &destinationPath)
    if err != nil {
        return
    }

    // 2. Build and start the command
    cmd := exec.Command("yt-dlp",
        "-x",
        "--audio-format", "mp3",
        "--newline",
        "-o", destinationPath+"/%(title)s.%(ext)s",
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

    // 3. Read progress
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

    // 4. Wait for finish
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