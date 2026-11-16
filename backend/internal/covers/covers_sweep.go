package covers

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

/*
MEDIA_ROOT/Covers holds two kinds of file: a media item's cover, named by the item's id, and a
playlist's or collection's image, named by its image id (copies share one). Files are removed
as soon as nothing uses them, but some slip through: a deleted user's collections go by cascade,
and a copy saved at the moment its original's image is removed can keep pointing at a file
that's gone. The sweep catches both
*/

// how old an unused file must be before the sweep removes it. Uploads write the file before
// the row that points at it is saved, so a younger one may be on its way into use
const sweepGrace = time.Hour

// how often the sweep runs after the first one at startup
const sweepEvery = 24 * time.Hour

// a *sql.DB or *sql.Tx
type rowQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
}

/*
Function:	RemoveUnusedImages
Purpose:	Delete the files of collection images no collection uses any more. Called after the
change that let go of them is saved; a file that can't be removed is only logged, since it no
longer shows anywhere
Params:
  - db: a *sql.DB or *sql.Tx
  - imageIDs: the image ids that may have been let go of
*/
func RemoveUnusedImages(db rowQuerier, imageIDs ...string) {
	for _, imageID := range imageIDs {
		var used bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM collections WHERE image_id = $1)`, imageID).Scan(&used); err != nil {
			logger.Error("checking whether collection image %s is still used: %s", imageID, err.Error())
			continue
		}
		if used {
			continue
		}
		if err := Remove(imageID); err != nil {
			logger.Error("removing collection image %s: %s", imageID, err.Error())
		}
	}
}

// a file in MEDIA_ROOT/Covers, as the sweep sees it
type coverFile struct {
	name    string
	dir     bool
	modTime time.Time
}

/*
Function:	orphans
Purpose:	The files the sweep removes: UUID-named files nothing references that are older than
the grace period
Params:
  - files: what's in MEDIA_ROOT/Covers
  - referenced: media item ids and collection image ids in use
  - now: the time to measure age against
*/
func orphans(files []coverFile, referenced map[string]bool, now time.Time) []string {
	var out []string
	for _, f := range files {
		if f.dir || !utils.IsUUID(f.name) || referenced[f.name] {
			continue
		}
		if now.Sub(f.modTime) < sweepGrace {
			continue
		}
		out = append(out, f.name)
	}
	return out
}

/*
Function:	missing
Purpose:	The image ids whose file isn't in MEDIA_ROOT/Covers
*/
func missing(imageIDs []string) []string {
	var out []string
	for _, id := range imageIDs {
		path, ok := Path(id)
		if !ok {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			out = append(out, id)
		}
	}
	return out
}

// the strings one query returns
func queryStrings(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

/*
Function:	Sweep
Purpose:	remove cover and image files nothing uses, and take away collection images whose
file is gone, so those show the mosaic again
Params:
  - ctx: stops the sweep
  - db: the database

Returns:	how many files were removed, how many collection images were cleared
*/
func Sweep(ctx context.Context, db *sql.DB) (removed, cleared int, err error) {
	root := os.Getenv("MEDIA_ROOT")
	if root == "" {
		return 0, 0, nil
	}

	// without the folder (no covers yet, or the NAS isn't mounted) every image would look
	// missing, so nothing is touched
	entries, err := os.ReadDir(filepath.Join(root, "Covers"))
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}

	imageIDs, err := queryStrings(ctx, db, `SELECT DISTINCT image_id::text FROM collections WHERE image_id IS NOT NULL`)
	if err != nil {
		return 0, 0, err
	}
	for _, id := range missing(imageIDs) {
		res, err := db.ExecContext(ctx, `UPDATE collections SET image_id = NULL, updated_at = NOW() WHERE image_id = $1`, id)
		if err != nil {
			return removed, cleared, err
		}
		n, _ := res.RowsAffected()
		cleared += int(n)
	}

	files := make([]coverFile, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, coverFile{name: e.Name(), dir: e.IsDir(), modTime: info.ModTime()})
	}

	// read after listing the files, so anything saved in between counts as used
	used, err := queryStrings(ctx, db, `
		SELECT id::text FROM media_items
		UNION SELECT image_id::text FROM collections WHERE image_id IS NOT NULL`)
	if err != nil {
		return removed, cleared, err
	}
	referenced := make(map[string]bool, len(used))
	for _, id := range used {
		referenced[id] = true
	}

	for _, name := range orphans(files, referenced, time.Now()) {
		if err := Remove(name); err != nil {
			logger.Error("removing unused cover %s: %s", name, err.Error())
			continue
		}
		removed++
	}
	return removed, cleared, nil
}

/*
Function:	StartSweeps
Purpose:	run Sweep now and then once a day, logging what it did
*/
func StartSweeps(ctx context.Context, db *sql.DB) {
	for {
		removed, cleared, err := Sweep(ctx, db)
		if err != nil {
			logger.Warn("sweeping unused covers: %v", err)
		}
		if removed > 0 || cleared > 0 {
			logger.Info("removed %d unused cover file(s), cleared %d collection image(s) whose file was gone", removed, cleared)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(sweepEvery):
		}
	}
}
