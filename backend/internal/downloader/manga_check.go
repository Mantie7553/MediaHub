package downloader

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// ErrCheckRunning: the manga already has a download job queued or running
var ErrCheckRunning = errors.New("this manga is already downloading")

// MangaDest is the folder manga is downloaded into
func MangaDest() string {
	return filepath.Join(os.Getenv("MEDIA_ROOT"), "Manga")
}

/*
Function:	NewChapterCheck
Purpose:	Record a download job that fetches the chapters a manga's sources have that the
library doesn't. The caller runs it with RunMangaDex. A job belongs to a request, so the
manga's latest one is reused, or one is recorded for this check

Params:
  - db: a pointer to our database
  - mediaItemID: the manga
  - requestedBy: the admin who asked, nil for the nightly check
*/
func NewChapterCheck(db *sql.DB, mediaItemID string, requestedBy *string) (string, error) {
	var running bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM download_jobs WHERE media_item_id = $1 AND status IN ('queued', 'downloading'))`,
		mediaItemID,
	).Scan(&running)
	if err != nil {
		return "", err
	}
	if running {
		return "", ErrCheckRunning
	}

	var requestID string
	err = db.QueryRow(
		`SELECT id FROM download_requests WHERE media_item_id = $1 ORDER BY requested_at DESC LIMIT 1`, mediaItemID,
	).Scan(&requestID)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.QueryRow(
			`INSERT INTO download_requests (requested_by, media_item_id, source_url, status, auto_approved, resolved_at)
			VALUES ($1, $2, '', 'approved', TRUE, NOW())
			RETURNING id`,
			requestedBy, mediaItemID,
		).Scan(&requestID)
	}
	if err != nil {
		return "", err
	}

	var jobID string
	err = db.QueryRow(
		`INSERT INTO download_jobs (request_id, media_item_id, source_url, destination_path, handler)
		VALUES ($1, $2, '', $3, 'mangadex')
		RETURNING id`,
		requestID, mediaItemID, MangaDest(),
	).Scan(&jobID)
	return jobID, err
}

// only one nightly pass at a time, a slow one could still be going the next night
var checkingLibrary atomic.Bool

/*
Function:	CheckLibraryForNewChapters
Purpose:	Look for new chapters of every manga linked to MangaDex or a Suwayomi source, and
download them. A job is only recorded for a manga that has something missing, so the
Downloads page isn't filled with empty checks, and the jobs run one after another to stay
within MangaDex's and the source sites' rate limits
*/
func CheckLibraryForNewChapters(db *sql.DB) {
	if !checkingLibrary.CompareAndSwap(false, true) {
		logger.Info("New chapter check skipped, the last one is still running")
		return
	}
	defer checkingLibrary.Store(false)

	rows, err := db.Query(
		`SELECT id FROM media_items mi
		WHERE type = 'manga' AND (
			(external_source = 'mangadex' AND COALESCE(external_id, '') <> '')
			OR EXISTS (SELECT 1 FROM suwayomi_items s WHERE s.media_item_id = mi.id))`,
	)
	if err != nil {
		logger.Error("new chapter check: %s", err.Error())
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	started := 0
	for _, id := range ids {
		plan, err := listChapters(db, id)
		if err != nil {
			continue
		}
		have, err := ChapterNumbers(db, id)
		if err != nil {
			continue
		}
		// a stale link finds nothing new, so no job (and no note on the Downloads page) says so
		if plan.link != nil && plan.swErr == nil {
			if note := staleLinkNote(plan.link.SourceName, plan.swLatest, knownLatest(db, id, plan.chapters, have)); note != "" {
				logger.Warn("%s: %s", plan.title, note[2:])
			}
		}
		missing := false
		for _, ch := range plan.chapters {
			if !have[ch.Value] {
				missing = true
				break
			}
		}
		if !missing {
			continue
		}

		jobID, err := NewChapterCheck(db, id, nil)
		if err != nil {
			if !errors.Is(err, ErrCheckRunning) {
				logger.Error("new chapter check for %s: %s", plan.title, err.Error())
			}
			continue
		}
		started++
		RunMangaDex(db, jobID, id, MangaDest())
	}
	logger.Info("New chapter check done: %d of %d manga had new chapters", started, len(ids))
}
