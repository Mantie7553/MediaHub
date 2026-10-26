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
	CanEdit   bool `json:"can_edit"`
	CanDelete bool `json:"can_delete"`
	// a copy an admin approved, which everyone sees and nobody changes
	Featured bool `json:"featured"`
	// a submission to feature it is waiting for an admin
	InReview bool `json:"in_review"`
	// for the owner: their latest request to feature it, and the featured copy of it if
	// there is one, so a new request reads as an update to that
	Submission     *SubmissionState `json:"submission"`
	FeaturedCopyID *string          `json:"featured_copy_id"`
	Items          []Entry          `json:"items"`
}

// SubmissionState is where a request to feature a playlist or collection has got to
type SubmissionState struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	AdminNote   *string    `json:"admin_note"`
	SubmittedAt time.Time  `json:"submitted_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
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
	// one season of an anime, rather than the whole show
	SeasonNumber *int `json:"season_number"`
	// albums (their downloaded tracks), nested playlists (their entries) and seasons (their
	// downloaded episodes)
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
	// with an anime's media_item_id: just that season of it
	SeasonNumber *int `json:"season_number"`
}

type addResponse struct {
	Added   int `json:"added"`
	Already int `json:"already"`
}

// After is the entry to move it below, or null for the top
type moveRequest struct {
	After *string `json:"after"`
}

type submitRequest struct {
	Note *string `json:"note"`
}

// PendingSubmission is a request to feature a playlist or collection, as admins review it
type PendingSubmission struct {
	ID          string    `json:"id"`
	Note        *string   `json:"note"`
	SubmittedAt time.Time `json:"submitted_at"`
	SubmittedBy string    `json:"submitted_by"`
	Collection  Summary   `json:"collection"`
	// the featured copy approving it would update, when one was made from it before
	UpdatesID   *string `json:"updates_id"`
	UpdatesName *string `json:"updates_name"`
}
