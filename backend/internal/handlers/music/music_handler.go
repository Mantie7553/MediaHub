package music

import (
	"database/sql"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
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
	Tracks      []AlbumTrack `json:"tracks"`
}

type AlbumTrack struct {
	MediaItemID  string  `json:"media_item_id"`
	Title        string  `json:"title"`
	TrackNumber  *int    `json:"track_number"`
	DurationSecs *int    `json:"duration_secs"`
	FilePath     *string `json:"file_path"`
}

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
			COALESCE(a.cover_image_url, (
				SELECT mi.cover_image_url FROM music_metadata mm2
				JOIN media_items mi ON mi.id = mm2.media_item_id
				WHERE mm2.album_id = a.id AND mi.cover_image_url IS NOT NULL
				LIMIT 1
			)) as cover_image_url,
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
	albumID := chi.URLParam(r, "id")

	var album AlbumDetail
	err := h.db.QueryRow(
		`SELECT id, title, artist, cover_image_url, release_date FROM albums WHERE id = $1`,
		albumID,
	).Scan(&album.ID, &album.Title, &album.Artist, &album.CoverURL, &album.ReleaseDate)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "album not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	rows, err := h.db.Query(`
		SELECT mm.media_item_id, mi.title, mm.track_number, mm.duration_secs, mm.file_path
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
		if err := rows.Scan(&t.MediaItemID, &t.Title, &t.TrackNumber, &t.DurationSecs, &t.FilePath); err != nil {
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
