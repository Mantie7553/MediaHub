package lists

import "time"

type rateRequest struct {
	MediaItemId string  `json:"media_item_id"`
	AlbumId     *string `json:"album_id"`
	Status      string  `json:"status"`
	Rating      *int    `json:"rating"`
}

type updateRequest struct {
	Status string `json:"status"`
	Rating *int   `json:"rating"`
}

type UserMediaEntry struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	Rating        *int       `json:"rating"`
	UpdatedAt     time.Time  `json:"updated_at"`
	MediaItemID   *string    `json:"media_item_id"`
	MediaType     string     `json:"media_type"`
	MediaTitle    string     `json:"media_title"`
	CoverImageURL *string    `json:"cover_image_url"`
	ReleaseDate   *time.Time `json:"release_date"`
	Artist        *string    `json:"artist"`
	AlbumID       *string    `json:"album_id"`
	ActiveSeason  *int       `json:"active_season"`
	SeasonWatched *int       `json:"season_watched"`
	SeasonTotal   *int       `json:"season_total"`
	ExternalID    *string    `json:"external_id"`
}

type progressRequest struct {
	Watched bool `json:"watched"`
}

// ContinueEntry is a title the user started and hasn't finished, shaped like UserMediaEntry so
// the same cards show it
type ContinueEntry struct {
	ID             string     `json:"id"`
	MediaItemID    string     `json:"media_item_id"`
	MediaType      string     `json:"media_type"`
	MediaTitle     string     `json:"media_title"`
	CoverImageURL  *string    `json:"cover_image_url"`
	ReleaseDate    *time.Time `json:"release_date"`
	ExternalID     *string    `json:"external_id"`
	ExternalSource *string    `json:"external_source"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	ProgressPct    *int       `json:"progress_pct"`
	ProgressLabel  *string    `json:"progress_label"`
}
