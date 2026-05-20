package plex

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/plex"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db   *sql.DB
	plex *plex.PlexClient
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:   db,
		plex: plex.NewPlexClient("PLEX_URL", "PLEX_TOKEN"),
	}
}

func (h *Handler) GetLibraries(w http.ResponseWriter, r *http.Request) {
	libraries, err := h.plex.GetLibraries()
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, libraries)
}

func (h *Handler) Link(w http.ResponseWriter, r *http.Request) {
	mediaItemId := chi.URLParam(r, "id")

	var req linkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var plexItemID string
	err := h.db.QueryRow(
		`INSERT INTO plex_items (media_item_id, plex_rating_key, plex_library_id)
		VALUES ($1, $2, $3)
		RETURNING id`,
		mediaItemId, req.RatingKey, req.LibraryID,
	).Scan(&plexItemID)

	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, map[string]string{"id": plexItemID}, http.StatusCreated)
}

func (h *Handler) GetStreamURL(w http.ResponseWriter, r *http.Request) {
	mediaItemId := chi.URLParam(r, "id")

	var ratingKey string
	err := h.db.QueryRow(
		`SELECT plex_rating_key FROM plex_items WHERE media_item_id = $1`,
		mediaItemId,
	).Scan(&ratingKey)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "no plex entry for this item")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, streamResponse{URL: h.plex.StreamURL(ratingKey)})
}
