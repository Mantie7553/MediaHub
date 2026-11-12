package covers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

/*
Covers that come from somewhere else (Sonarr, Radarr, AniList, MangaDex, a title saved from
search) are downloaded once and kept with the uploaded ones in MEDIA_ROOT/Covers, so the library
never depends on another site serving, or allowing, the image. cover_origin_url remembers where
the copy came from, so a sync handing back the same address keeps the copy
*/

// MaxRemoteBytes is the largest cover downloaded from another site
const MaxRemoteBytes = 20 << 20

// covers downloaded at once by LocalizeAll
const localizeWorkers = 4

/*
Function:	ReadImage
Purpose:	read a cover download's body, refusing a failed response, anything that isn't an image
and anything over MaxRemoteBytes
*/
func ReadImage(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
		return nil, fmt.Errorf("cover fetch returned %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxRemoteBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxRemoteBytes || !strings.HasPrefix(http.DetectContentType(data), "image/") {
		return nil, errors.New("cover is too large or not an image")
	}
	return data, nil
}

/*
Function:	publicOnly
Purpose:	dialer check refusing addresses inside the server's own network. A cover's address can
come from a user saving a title, so it must not reach the database, Sonarr or the router
*/
func publicOnly(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !isPublic(ip) {
		return fmt.Errorf("cover address %s is not public", host)
	}
	return nil
}

// carrier-grade NAT, shared address space that isn't on the internet either
var sharedSpace = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

func isPublic(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || sharedSpace.Contains(ip))
}

/*
Function:	newClient
Purpose:	the client covers are downloaded with. check is run on every address dialled,
redirects included, and no proxy is used so the check sees the real address
*/
func newClient(check func(network, address string, c syscall.RawConn) error) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, Control: check}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy:               nil,
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConnsPerHost: localizeWorkers,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to %s not allowed", req.URL.Scheme)
			}
			return nil
		},
	}
}

var client = newClient(publicOnly)

/*
Function:	IsRemote
Purpose:	whether a cover_image_url is on another site, rather than served by MediaHub
*/
func IsRemote(coverURL string) bool {
	u, err := url.Parse(coverURL)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}

/*
Function:	Fetch
Purpose:	download a cover from another site
Params:
  - rawURL: an http or https address on a public host
*/
func Fetch(rawURL string) ([]byte, error) {
	return fetchWith(client, rawURL)
}

func fetchWith(c *http.Client, rawURL string) ([]byte, error) {
	if !IsRemote(rawURL) {
		return nil, fmt.Errorf("%q is not an http or https address", rawURL)
	}
	resp, err := c.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ReadImage(resp)
	if err != nil {
		return nil, err
	}
	// kept only in the formats ServeCover can send back
	if musicfiles.ImageType(data) == "" {
		return nil, errors.New("cover is not a JPEG, PNG, WebP or GIF")
	}
	return data, nil
}

/*
Function:	Localize
Purpose:	download a media item's cover from another site, keep it in MEDIA_ROOT/Covers and point
the item at it. Does nothing when the item's cover has changed from remoteURL meanwhile
Params:
  - db: the database
  - mediaItemID: the media item
  - remoteURL: its current cover_image_url
*/
func Localize(db *sql.DB, mediaItemID, remoteURL string) error {
	return localizeWith(db, client, mediaItemID, remoteURL)
}

func localizeWith(db *sql.DB, c *http.Client, mediaItemID, remoteURL string) error {
	path, ok := Path(mediaItemID)
	if !ok {
		return errors.New("MEDIA_ROOT is not set")
	}
	// nothing is downloaded for an item whose cover isn't remoteURL (anymore)
	var wanted bool
	err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM media_items WHERE id = $1 AND cover_image_url = $2)`, mediaItemID, remoteURL).Scan(&wanted)
	if err != nil || !wanted {
		return err
	}
	data, err := fetchWith(c, remoteURL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".download"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	defer os.Remove(tmp)

	// the row stays locked while the file is swapped in, so a second download of the same
	// cover waits and then finds there's nothing left to do
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current sql.NullString
	err = tx.QueryRow(`SELECT cover_image_url FROM media_items WHERE id = $1 FOR UPDATE`, mediaItemID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && current.String != remoteURL) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := musicfiles.WriteCover(path, data); err != nil {
		return err
	}
	cache.Disk().Forget(path)
	if _, err := tx.Exec(
		`UPDATE media_items SET cover_image_url = $1, cover_origin_url = $2 WHERE id = $3`,
		URL(mediaItemID), remoteURL, mediaItemID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

/*
Function:	LocalizeLater
Purpose:	Localize in the background, for a title just added, logging a failure. The nightly
sync tries again
*/
func LocalizeLater(db *sql.DB, mediaItemID, remoteURL string) {
	if !IsRemote(remoteURL) {
		return
	}
	go func() {
		if err := Localize(db, mediaItemID, remoteURL); err != nil {
			logger.Warn("saving the cover of %s: %v", mediaItemID, err)
		}
	}()
}

// one LocalizeAll at a time: startup and the sync can both start one
var localizeAllMu sync.Mutex

/*
Function:	LocalizeAll
Purpose:	download every cover still on another site. One that fails stays as it is and is
tried again next time
Returns:	how many were saved and how many failed
*/
func LocalizeAll(ctx context.Context, db *sql.DB) (saved, failed int, err error) {
	if !localizeAllMu.TryLock() {
		return 0, 0, nil
	}
	defer localizeAllMu.Unlock()
	if _, ok := Path("00000000-0000-0000-0000-000000000000"); !ok {
		return 0, 0, errors.New("MEDIA_ROOT is not set")
	}

	type job struct{ id, url string }
	rows, err := db.QueryContext(ctx, `SELECT id, cover_image_url FROM media_items WHERE cover_image_url ~ '^https?://'`)
	if err != nil {
		return 0, 0, err
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.url); err != nil {
			rows.Close()
			return 0, 0, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	queue := make(chan job)
	for range localizeWorkers {
		wg.Go(func() {
			for j := range queue {
				err := Localize(db, j.id, j.url)
				mu.Lock()
				if err != nil {
					failed++
					logger.Warn("saving the cover of %s from %s: %v", j.id, j.url, err)
				} else {
					saved++
				}
				mu.Unlock()
			}
		})
	}
	for _, j := range jobs {
		if ctx.Err() != nil {
			break
		}
		queue <- j
	}
	close(queue)
	wg.Wait()
	return saved, failed, ctx.Err()
}

/*
Function:	KeepLocal
Purpose:	SQL for the value a sync sets cover_image_url to: value, unless the item already has
a saved copy of that same cover, which it keeps. Goes inside locks.Keep
Params:
  - value: the SQL for the cover the sync has, such as $2
*/
func KeepLocal(value string) string {
	return fmt.Sprintf("CASE WHEN media_items.cover_origin_url = %s AND media_items.cover_image_url = '/api/media/' || media_items.id || '/cover' THEN media_items.cover_image_url ELSE %s END",
		value, value)
}
