package lists

import "time"

type rateRequest struct {
	MediaItemId string `json:"media_item_id"`
	Status      string `json:"status"`
	Rating      *int   `json:"rating"`
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
	MediaItemID   string     `json:"media_item_id"`
	MediaType     string     `json:"media_type"`
	MediaTitle    string     `json:"media_title"`
	CoverImageURL *string    `json:"cover_image_url"`
	ReleaseDate   *time.Time `json:"release_date"`
	Artist        *string    `json:"artist"`
	ActiveSeason  *int       `json:"active_season"`
	SeasonWatched *int       `json:"season_watched"`
	SeasonTotal   *int       `json:"season_total"`
	ExternalID    *string    `json:"external_id"`
}

type progressRequest struct {
	Watched bool `json:"watched"`
}
