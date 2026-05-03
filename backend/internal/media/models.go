package media

import "time"

type uploadRequest struct {
	Type           string `json:"type"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	CoverImageURL  string `json:"cover_image_url"`
	ReleaseDate    string `json:"release_date"`
	ExternalID     string `json:"external_id"`
	ExternalSource string `json:"external_source"`

	// anime
	Studio string   `json:"studio"`
	Status string   `json:"status"`
	Genres []string `json:"genres"`

	// movie
	RuntimeMins *int   `json:"runtime_mins"`
	Director    string `json:"director"`

	// music_track
	Artist      string `json:"artist"`
	TrackNumber *int   `json:"track_number"`
	DurationSec *int   `json:"duration_secs"`
}

type MediaItem struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Title          string     `json:"title"`
	Description    *string    `json:"description"`
	CoverImageURL  *string    `json:"cover_image_url"`
	ReleaseDate    *time.Time `json:"release_date"`
	ExternalID     *string    `json:"external_id"`
	ExternalSource *string    `json:"external_source"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AnimeMetadata struct {
	Studio *string  `json:"studio"`
	Status *string  `json:"status"`
	Genres []string `json:"genres"`
}

type MovieMetadata struct {
	RuntimeMins *int     `json:"runtime_mins"`
	Director    *string  `json:"director"`
	Genres      []string `json:"genres"`
}

type MusicMetadata struct {
	Artist       string   `json:"artist"`
	TrackNumber  *int     `json:"track_number"`
	DurationSecs *int     `json:"duration_secs"`
	Genres       []string `json:"genres"`
}

type MediaItemDetail struct {
	MediaItem
	Metadata any `json:"metadata"`
}
