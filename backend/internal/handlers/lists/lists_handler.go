package lists

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	Add
Purpose:	Add a new entry to the user_media_status table
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var req rateRequest
	user := auth.GetUser(r)

	// decode the incoming request, check that the structure is correct
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	// confirm the media item id and status have been provided
	if (req.MediaItemId == "" && req.AlbumId == nil) || req.Status == "" {
		utils.Error(w, http.StatusBadRequest, "status is required")
		return
	}
	if !validRating(req.Rating) {
		utils.Error(w, http.StatusBadRequest, "rating must be 1 to 10")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	// add the new entry in the database
	var statusId string
	err = tx.QueryRow(
		`INSERT INTO user_media_status (user_id, media_item_id, album_id, status, rating)
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`,
		user.UserID, utils.NullString(req.MediaItemId), req.AlbumId, req.Status, req.Rating,
	).Scan(&statusId)
	if err != nil {
		if utils.IsUniqueViolation(err) {
			utils.Error(w, http.StatusConflict, "status already exists")
			return
		}
		utils.InternalError(w, err)
		return
	}

	// a title added as completed has every episode, chapter or volume done too
	if req.Status == "completed" && req.MediaItemId != "" {
		if utils.InternalError(w, tracking.CompleteAll(tx, user.UserID, req.MediaItemId)) {
			return
		}
	} else if req.Progress != nil && req.MediaItemId != "" {
		if utils.InternalError(w, tracking.SetProgressCount(tx, user.UserID, req.MediaItemId, *req.Progress)) {
			return
		}
	}

	if utils.InternalError(w, tx.Commit()) {
		return
	}

	// return the id of the new entry
	utils.JSON(w, map[string]string{"id": statusId}, http.StatusCreated)
}

/*
Function:	GetAll
Purpose:	Get all of the media currently in the database
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)

	items := []UserMediaEntry{}

	queryString := `
	SELECT ums.id, ums.status, ums.rating, ums.updated_at, ums.album_id,
		mi.id, mi.type, mi.title, mi.cover_image_url, mi.release_date, mi.external_id, mi.external_source,
		mm.artist,
		anime_progress.active_season, anime_progress.season_watched, anime_progress.season_total,
		-- how far through its parts: the files done, or the count given when that's further
		GREATEST(parts.done, COALESCE(ums.progress_count, 0)),
		GREATEST(parts.files, parts.source), parts.source > 0
	FROM user_media_status ums
	JOIN media_items mi ON mi.id = ums.media_item_id
	LEFT JOIN music_metadata mm ON mm.media_item_id = mi.id
	LEFT JOIN LATERAL (
		SELECT 
			e.season_number AS active_season,
			COUNT(*) FILTER (WHERE uap.watched = true) AS season_watched,
			COUNT(*) AS season_total
		FROM episodes e
		LEFT JOIN user_anime_progress uap ON uap.episode_id = e.id AND uap.user_id = ums.user_id
		WHERE e.media_item_id = mi.id
		-- the season shown: the last one once every episode is watched, otherwise the season of
		-- the latest episode watched (the later season when a whole run was marked at once)
		AND e.season_number = COALESCE(
			(SELECT MAX(e3.season_number) FROM episodes e3
			LEFT JOIN user_anime_progress uap3 ON uap3.episode_id = e3.id AND uap3.user_id = ums.user_id
			WHERE e3.media_item_id = mi.id
			HAVING COUNT(*) = COUNT(*) FILTER (WHERE uap3.watched)),
			(SELECT e2.season_number FROM episodes e2
			JOIN user_anime_progress uap2 ON uap2.episode_id = e2.id
			WHERE e2.media_item_id = mi.id AND uap2.user_id = ums.user_id AND uap2.watched = true
			ORDER BY uap2.watched_at DESC, e2.season_number DESC, e2.episode_number DESC LIMIT 1),
			1
		)
		GROUP BY e.season_number
	) anime_progress ON true
	-- episodes, chapters or volumes: done, the files there are, and the source's total (Sonarr
	-- lists every episode, so an anime's rows are its total). Matches tracking.partsQuery
	LEFT JOIN LATERAL (
		SELECT COUNT(*) FILTER (WHERE p.watched) AS done, COUNT(*) AS files, COUNT(*) AS source
		FROM episodes e
		LEFT JOIN user_anime_progress p ON p.episode_id = e.id AND p.user_id = ums.user_id
		WHERE e.media_item_id = mi.id
		HAVING mi.type = 'anime'
		UNION ALL
		SELECT COUNT(*) FILTER (WHERE p.completed), COUNT(*),
			COALESCE((SELECT total_chapters FROM manga_metadata WHERE media_item_id = mi.id), 0)
		FROM manga_chapters c
		LEFT JOIN manga_progress p ON p.chapter_id = c.id AND p.user_id = ums.user_id
		WHERE c.media_item_id = mi.id
		HAVING mi.type = 'manga'
		UNION ALL
		SELECT COUNT(*) FILTER (WHERE p.completed), COUNT(*),
			COALESCE((SELECT total_volumes FROM light_novel_metadata WHERE media_item_id = mi.id), 0)
		FROM light_novel_volumes v
		LEFT JOIN light_novel_progress p ON p.volume_id = v.id AND p.user_id = ums.user_id
		WHERE v.media_item_id = mi.id
		HAVING mi.type = 'light_novel'
	) parts ON mi.type IN ('anime', 'manga', 'light_novel')
	WHERE ums.user_id = $1 AND ums.media_item_id IS NOT NULL

	UNION ALL

	SELECT ums.id, ums.status, ums.rating, ums.updated_at, ums.album_id,
		NULL, 'music_album', a.title, a.cover_image_url, NULL, NULL, NULL,
		a.artist,
		NULL, NULL, NULL,
		NULL, NULL, NULL
	FROM user_media_status ums
	JOIN albums a ON a.id = ums.album_id
	WHERE ums.user_id = $1 AND ums.album_id IS NOT NULL`

	rows, err := h.db.Query(queryString, user.UserID)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item UserMediaEntry
		err := rows.Scan(
			&item.ID, &item.Status, &item.Rating, &item.UpdatedAt, &item.AlbumID,
			&item.MediaItemID, &item.MediaType, &item.MediaTitle, &item.CoverImageURL,
			&item.ReleaseDate, &item.ExternalID, &item.ExternalSource, &item.Artist,
			&item.ActiveSeason, &item.SeasonWatched, &item.SeasonTotal,
			&item.ProgressCount, &item.ProgressTotal, &item.ProgressTotalKnown,
		)
		if utils.InternalError(w, err) {
			return
		}
		if utils.InternalError(w, auth.SignCoverURL(item.CoverImageURL, user.UserID)) {
			return
		}
		items = append(items, item)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, items)
}

/*
Function:	Update
Purpose:	Handle updating the status for a given media item for a specific user
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	// get the user information from the request
	user := auth.GetUser(r)
	// get the entries id from the URL parameters
	entryID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	// decode the incoming request, check that the structure is correct
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if req.Status.Set && (req.Status.Value == nil || *req.Status.Value == "") {
		utils.Error(w, http.StatusBadRequest, "status can't be empty")
		return
	}
	if req.Rating.Set && !validRating(req.Rating.Value) {
		utils.Error(w, http.StatusBadRequest, "rating must be 1 to 10")
		return
	}
	// nil keeps the saved status (the column is an enum, so "" can't stand in for "unchanged")
	status := req.Status.Value

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	// do the update; a status or rating left out keeps the one saved
	var mediaStatusId string
	var mediaItemID sql.NullString
	err = tx.QueryRow(
		`UPDATE user_media_status
		SET status = COALESCE($1, status),
			rating = CASE WHEN $2 THEN $3 ELSE rating END,
			updated_at = NOW()
		WHERE id = $4 AND user_id = $5
		RETURNING id, media_item_id`,
		status, req.Rating.Set, req.Rating.Value, entryID, user.UserID,
	).Scan(&mediaStatusId, &mediaItemID)

	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	// completing a title completes every episode, chapter or volume; other statuses leave
	// progress as it is unless a count was picked. The status picked stands either way
	if status != nil && *status == "completed" && mediaItemID.Valid {
		if utils.InternalError(w, tracking.CompleteAll(tx, user.UserID, mediaItemID.String)) {
			return
		}
	} else if req.Progress != nil && mediaItemID.Valid {
		if utils.InternalError(w, tracking.SetProgressCount(tx, user.UserID, mediaItemID.String, *req.Progress)) {
			return
		}
	}

	if utils.InternalError(w, tx.Commit()) {
		return
	}

	// return the id for the updated status
	utils.JSON(w, map[string]string{"id": mediaStatusId})
}

/*
Function:	Delete
Purpose:	Remove a users status for a media item
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	// get the user information from the request
	user := auth.GetUser(r)
	// get the entries is from the URL parameters
	entryID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	// do the delete
	result, err := h.db.Exec(
		`DELETE FROM user_media_status WHERE id = $1 AND user_id = $2`,
		entryID, user.UserID,
	)

	if utils.InternalError(w, err) {
		return
	}

	// check that it was actually deleted
	rows, _ := result.RowsAffected()
	if rows == 0 {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	// return no content
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	UpdateProgress
Purpose:	Update a users progress for a specific piece of media
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	// get the user info from the request
	user := auth.GetUser(r)
	// get the entries id from the URL parameters
	episodeID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	// decode the incoming request, check that the structure is correct
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	// do the update
	var progressID string
	err := h.db.QueryRow(
		`INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, episode_id) 
		DO UPDATE SET watched = $3, watched_at = NOW()
		RETURNING id`,
		user.UserID, episodeID, req.Watched,
	).Scan(&progressID)

	if utils.InternalError(w, err) {
		return
	}

	if utils.InternalError(w, tracking.SyncStatusOf(h.db, user.UserID, tracking.EpisodeTitle, episodeID)) {
		return
	}

	// return the id
	utils.JSON(w, map[string]any{
		"id": progressID,
	})
}

func (h *Handler) MarkSeasonWatched(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	seasonNumber := chi.URLParam(r, "seasonNumber")

	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	_, err := h.db.Exec(
		`INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at) 
		SELECT $1, e.id, $2, NOW() 
		FROM episodes e 
		WHERE e.media_item_id = $3 AND e.season_number = $4 
		ON CONFLICT (user_id, episode_id) 
		DO UPDATE SET watched = $2, watched_at = NOW()`,
		user.UserID, req.Watched, id, seasonNumber,
	)

	if utils.InternalError(w, err) {
		return
	}

	if utils.InternalError(w, tracking.SyncStatus(h.db, user.UserID, id)) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkShowWatched(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	_, err := h.db.Exec(
		`INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at) 
		SELECT $1, e.id, $2, NOW() 
		FROM episodes e 
		WHERE e.media_item_id = $3
		ON CONFLICT (user_id, episode_id) 
		DO UPDATE SET watched = $2, watched_at = NOW()`,
		user.UserID, req.Watched, id,
	)

	if utils.InternalError(w, err) {
		return
	}

	if utils.InternalError(w, tracking.SyncStatus(h.db, user.UserID, id)) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	GetProgress
Purpose:	How far the user is through a title's episodes, chapters or volumes and how many it
has, for the progress picker; the title doesn't have to be on their list. 404 for titles
without parts
Params:
  - w: http response writer to respond to the front end
  - r: http request with the media item id in the URL
*/
func (h *Handler) GetProgress(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	count, parts, has, err := tracking.ProgressOf(h.db, user.UserID, id)
	if utils.InternalError(w, err) {
		return
	}
	if !has {
		utils.Error(w, http.StatusNotFound, "this title has no episodes, chapters or volumes")
		return
	}
	utils.JSON(w, map[string]any{
		"progress_count":       count,
		"progress_total":       parts.Total(),
		"progress_total_known": parts.Source > 0,
	})
}
