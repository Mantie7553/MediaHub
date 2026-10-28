package music

import (
	"database/sql"
	"errors"
	"io"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/lib/pq"
)

type AlbumSummary struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	CoverURL   *string `json:"cover_image_url"`
	TrackCount int     `json:"track_count"`
}

type AlbumDetail struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Artist      string       `json:"artist"`
	CoverURL    *string      `json:"cover_image_url"`
	ReleaseDate *string      `json:"release_date"`
	Description *string      `json:"description"`
	Tracks      []AlbumTrack `json:"tracks"`
}

type AlbumTrack struct {
	MediaItemID  string  `json:"media_item_id"`
	Title        string  `json:"title"`
	Artist       string  `json:"artist"`
	TrackNumber  *int    `json:"track_number"`
	DurationSecs *int    `json:"duration_secs"`
	FilePath     *string `json:"file_path"`
}

// an album's cover, or when it has none a cover from one of its tracks (a track saved from
// YouTube carries its thumbnail), preferring covers from the album's own folder.
// Used as a column wherever albums are listed or shown, so every view agrees
const albumCover = `COALESCE(a.cover_image_url, (
	SELECT mi.cover_image_url FROM music_metadata mm2
	JOIN media_items mi ON mi.id = mm2.media_item_id
	WHERE mm2.album_id = a.id AND mi.cover_image_url IS NOT NULL
	ORDER BY mi.cover_image_url LIKE '/api/%' DESC, mm2.track_number NULLS LAST
	LIMIT 1
))`

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	GetAlbums
Purpose:	Get all albums that have at least one available track
Params:
  - w: http response writer
  - r: http request
*/
func (h *Handler) GetAlbums(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT a.id, a.title, a.artist,
			` + albumCover + ` AS cover_image_url,
			COUNT(mm.media_item_id) as track_count
		FROM albums a
		JOIN music_metadata mm ON mm.album_id = a.id AND mm.file_path IS NOT NULL
		GROUP BY a.id
		ORDER BY a.title`)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	albums := []AlbumSummary{}
	for rows.Next() {
		var a AlbumSummary
		if err := rows.Scan(&a.ID, &a.Title, &a.Artist, &a.CoverURL, &a.TrackCount); err != nil {
			utils.InternalError(w, err)
			return
		}
		if utils.InternalError(w, auth.SignCoverURL(a.CoverURL, auth.GetUser(r).UserID)) {
			return
		}
		albums = append(albums, a)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, albums)
}

/*
Function:	GetAlbum
Purpose:	Get a single album with its full track listing
Params:
  - w: http response writer
  - r: http request with URL param {id}
*/
func (h *Handler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	albumID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var album AlbumDetail
	err := h.db.QueryRow(
		`SELECT a.id, a.title, a.artist, `+albumCover+`, a.release_date, a.description FROM albums a WHERE a.id = $1`,
		albumID,
	).Scan(&album.ID, &album.Title, &album.Artist, &album.CoverURL, &album.ReleaseDate, &album.Description)

	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "album not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	if utils.InternalError(w, auth.SignCoverURL(album.CoverURL, auth.GetUser(r).UserID)) {
		return
	}

	rows, err := h.db.Query(`
		SELECT mm.media_item_id, mi.title, mm.artist, mm.track_number, mm.duration_secs, mm.file_path
		FROM music_metadata mm
		JOIN media_items mi ON mi.id = mm.media_item_id
		WHERE mm.album_id = $1
		ORDER BY mm.track_number, mi.title`,
		albumID,
	)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	album.Tracks = []AlbumTrack{}
	for rows.Next() {
		var t AlbumTrack
		if err := rows.Scan(&t.MediaItemID, &t.Title, &t.Artist, &t.TrackNumber, &t.DurationSecs, &t.FilePath); err != nil {
			utils.InternalError(w, err)
			return
		}
		album.Tracks = append(album.Tracks, t)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, album)
}

/*
Function:	GetRecommended
Purpose:	Get albums from artists the user has saved that they haven't added to their list yet
Params:
  - w: http response writer
  - r: http request
*/
func (h *Handler) GetRecommended(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)

	rows, err := h.db.Query(`
		SELECT a.id, a.title, a.artist,
			`+albumCover+` AS cover_image_url,
			COUNT(mm.media_item_id) as track_count
		FROM albums a
		JOIN music_metadata mm ON mm.album_id = a.id AND mm.file_path IS NOT NULL
		WHERE a.artist IN (
			SELECT DISTINCT a2.artist
			FROM user_media_status ums
			JOIN albums a2 ON a2.id = ums.album_id
			WHERE ums.user_id = $1
		)
		AND a.id NOT IN (
			SELECT album_id FROM user_media_status
			WHERE user_id = $1 AND album_id IS NOT NULL
		)
		GROUP BY a.id
		ORDER BY a.title`,
		user.UserID,
	)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	albums := []AlbumSummary{}
	for rows.Next() {
		var a AlbumSummary
		if err := rows.Scan(&a.ID, &a.Title, &a.Artist, &a.CoverURL, &a.TrackCount); err != nil {
			utils.InternalError(w, err)
			return
		}
		if utils.InternalError(w, auth.SignCoverURL(a.CoverURL, auth.GetUser(r).UserID)) {
			return
		}
		albums = append(albums, a)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, albums)
}

// largest cover image accepted, well above a high resolution album scan
const maxCoverBytes = 10 << 20

/*
Function:	UploadCover
Purpose:	Admin: replace an album's cover. The image is saved as cover.jpg in the folder of the
album's first downloaded track, the cover every track in that folder serves, and becomes
the album's cover

Params:
  - w: http response writer
  - r: multipart form with "file", a JPEG, PNG, WebP or GIF image
*/
func (h *Handler) UploadCover(w http.ResponseWriter, r *http.Request) {
	albumID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var trackID, trackPath string
	err := h.db.QueryRow(
		`SELECT mm.media_item_id, mm.file_path FROM music_metadata mm
		WHERE mm.album_id = $1 AND mm.file_path IS NOT NULL
		ORDER BY mm.track_number NULLS LAST
		LIMIT 1`,
		albumID,
	).Scan(&trackID, &trackPath)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "album not found or none of its tracks are downloaded")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCoverBytes+(1<<20))
	if err := r.ParseMultipartForm(maxCoverBytes); err != nil {
		utils.Error(w, http.StatusRequestEntityTooLarge, "the image is too large or the upload was cut off")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "no image in the upload")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxCoverBytes+1))
	if utils.InternalError(w, err) {
		return
	}
	if len(data) > maxCoverBytes {
		utils.Error(w, http.StatusRequestEntityTooLarge, "the image is too large")
		return
	}
	if musicfiles.ImageType(data) == "" {
		utils.Error(w, http.StatusBadRequest, "covers must be JPEG, PNG, WebP or GIF images")
		return
	}

	coverPath := musicfiles.CoverPath(trackPath)
	if err := musicfiles.WriteCover(coverPath, data); err != nil {
		logger.Error("saving the cover for album %s: %s", albumID, err.Error())
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	// the cached copy of the old cover is never served again
	cache.Disk().Forget(coverPath)

	coverURL := musicfiles.CoverURL(trackID)
	_, err = h.db.Exec(`UPDATE albums SET cover_image_url = $1 WHERE id = $2`, coverURL, albumID)
	if utils.InternalError(w, err) {
		return
	}

	if utils.InternalError(w, auth.SignCoverURL(&coverURL, auth.GetUser(r).UserID)) {
		return
	}
	utils.JSON(w, map[string]string{"cover_image_url": coverURL})
}

type ArtistSummary struct {
	Name   string   `json:"name"`
	Albums []string `json:"albums"`
}

/*
Function:	GetArtists
Purpose:	every artist in the library with their album titles, for suggestions when naming
the artist and album of a new download
Params:
  - w: http response writer
  - r: http request
*/
func (h *Handler) GetArtists(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT x.artist,
			COALESCE(array_agg(DISTINCT a.title) FILTER (WHERE a.title IS NOT NULL), '{}')
		FROM (
			SELECT artist FROM music_metadata WHERE artist <> ''
			UNION
			SELECT artist FROM albums WHERE artist <> ''
		) x
		LEFT JOIN albums a ON a.artist = x.artist
		GROUP BY x.artist
		ORDER BY LOWER(x.artist)`)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	artists := []ArtistSummary{}
	for rows.Next() {
		var a ArtistSummary
		if err := rows.Scan(&a.Name, pq.Array(&a.Albums)); err != nil {
			utils.InternalError(w, err)
			return
		}
		artists = append(artists, a)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, artists)
}

/*
Function:	RecordPlay
Purpose:	count a play of a track for the current user. The player calls this once someone has
listened long enough, so skipping through a queue isn't counted
Params:
  - w: http response writer
  - r: http request, with the track's media item id in the path
*/
func (h *Handler) RecordPlay(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	user := auth.GetUser(r)

	res, err := h.db.Exec(
		`INSERT INTO music_plays (user_id, media_item_id)
		SELECT $1, media_item_id FROM music_metadata WHERE media_item_id = $2`,
		user.UserID, id,
	)
	if utils.InternalError(w, err) {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		utils.Error(w, http.StatusNotFound, "track not found")
		return
	}
	auth.MarkActiveToday(h.db, user.UserID)
	w.WriteHeader(http.StatusNoContent)
}
