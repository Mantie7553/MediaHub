package lists

import "time"

type rateRequest struct {
	MediaItemId string `json:"media_item_id"`
	Status      string `json:"status"`
	Rating      *int   `json:"rating"`
	Notes       string `json:"notes"`
}

type updateRequest struct {
	Status string `json:"status"`
	Rating *int   `json:"rating"`
	Notes  string `json:"notes"`
}

type UserMediaEntry struct {
	ID              string     `json:"id"`
	Status          string     `json:"status"`
	Rating          *int       `json:"rating"`
	Notes           *string    `json:"notes"`
	UpdatedAt       time.Time  `json:"updated_at"`
	MediaItemID     string     `json:"media_item_id"`
	MediaType       string     `json:"media_type"`
	MediaTitle      string     `json:"media_title"`
	CoverImageURL   *string    `json:"cover_image_url"`
	ReleaseDate     *time.Time `json:"release_date"`
	Artist          *string    `json:"artist"`
	EpisodesWatched *int       `json:"episodes_watched"`
	SeasonID        *string    `json:"season_id"`
}

type progressRequest struct {
	EpisodesWatched int    `json:"episodes_watched"`
	SeasonID        string `json:"season_id"`
}
