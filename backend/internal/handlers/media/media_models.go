package media

import (
	"time"
)

// GENERAL
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

	// manga
	TotalChapters *int `json:"total_chapters"`
}

type progressRequest struct {
	LastPageRead int  `json:"last_page_read"`
	Completed    bool `json:"completed"`
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
	Artist         *string    `json:"artist"`
}

type MediaItemDetail struct {
	MediaItem
	Metadata any `json:"metadata"`
}

// ANIME
type AnimeMetadata struct {
	Studio *string  `json:"studio"`
	Status *string  `json:"status"`
	Genres []string `json:"genres"`
}

type Episode struct {
	ID            string  `json:"id"`
	SeasonNumber  int     `json:"season_number"`
	EpisodeNumber int     `json:"episode_number"`
	Title         *string `json:"title"`
}

// LIGHT NOVEL
type LightNovelMetadata struct {
	Author       *string  `json:"author"`
	TotalVolumes *int     `json:"total_volumes"`
	Genres       []string `json:"genres"`
}

type LightNovelVolume struct {
	ID           string  `json:"id"`
	VolumeNumber int     `json:"volume_number"`
	Title        *string `json:"title"`
}

type LightNovelDetail struct {
	LightNovelMetadata
	Volumes []LightNovelVolume `json:"volumes"`
}

// MANGA
type MangaMetadata struct {
	MediaItemID   string   `json:"media_item_id"`
	TotalChapters *int     `json:"total_chapters"`
	Genres        []string `json:"genres"`
	Status        *string  `json:"status"`
}

type MangaChapter struct {
	ID            string     `json:"id"`
	MediaItemID   string     `json:"media_item_id"`
	ChapterNumber float64    `json:"chapter_number"`
	Title         *string    `json:"title"`
	FilePath      *string    `json:"file_path"`
	PageCount     *int       `json:"page_count"`
	CreatedAt     *time.Time `json:"created_at"`
}

type MangaProgress struct {
	UserID       string    `json:"user_id"`
	ChapterID    string    `json:"chapter_id"`
	MediaItemID  string    `json:"media_item_id"`
	LastPageRead *string   `json:"last_page_read"`
	Completed    bool      `json:"completed"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type MangaDetail struct {
	MangaMetadata
	Chapters []MangaChapter `json:"chapters"`
}

// MOVIE
type MovieMetadata struct {
	RuntimeMins *int     `json:"runtime_mins"`
	Director    *string  `json:"director"`
	Genres      []string `json:"genres"`
	FilePath    *string  `json:"file_path"`
}

// MUSIC
type MusicMetadata struct {
	Artist       string   `json:"artist"`
	TrackNumber  *int     `json:"track_number"`
	DurationSecs *int     `json:"duration_secs"`
	Genres       []string `json:"genres"`
}
