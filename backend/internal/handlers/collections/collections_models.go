package collections

import "time"

// Summary is a playlist or collection as the dashboard rows and the "Add to…" picker show it
type Summary struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	// one of the computed playlists, which can be played but not changed
	Smart     bool `json:"smart"`
	ItemCount int  `json:"item_count"`
	// up to 4 covers from the start of it, for a mosaic
	Covers []string `json:"covers"`
	// it already holds the item asked about with ?contains=
	Contains  bool       `json:"contains"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// Detail is a playlist or collection with everything in it, in order
type Detail struct {
	Summary
	CanEdit bool    `json:"can_edit"`
	Items   []Entry `json:"items"`
}

/*
Entry is one thing in a playlist or collection. Type is the media type for tracks and titles
("music_track", "anime", ...), "album", or "playlist" for a nested one, and ID is that thing's
own id
*/
type Entry struct {
	// the entry itself, for removing or moving it; empty in a computed playlist
	EntryID        string     `json:"entry_id"`
	Type           string     `json:"type"`
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Artist         *string    `json:"artist"`
	CoverImageURL  *string    `json:"cover_image_url"`
	ReleaseDate    *time.Time `json:"release_date"`
	ExternalSource *string    `json:"external_source"`
	// tracks
	DurationSecs *int    `json:"duration_secs"`
	Available    bool    `json:"available"`
	AlbumID      *string `json:"album_id"`
	AlbumTitle   *string `json:"album_title"`
	// albums (their downloaded tracks) and nested playlists (their entries)
	ItemCount *int `json:"item_count"`
	// nested playlists
	Covers []string `json:"covers,omitempty"`
}

type createRequest struct {
	Kind        string  `json:"kind"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// exactly one of the three
type addRequest struct {
	MediaItemID  string `json:"media_item_id"`
	AlbumID      string `json:"album_id"`
	CollectionID string `json:"collection_id"`
}

type addResponse struct {
	Added   int `json:"added"`
	Already int `json:"already"`
}

// After is the entry to move it below, or null for the top
type moveRequest struct {
	After *string `json:"after"`
}
