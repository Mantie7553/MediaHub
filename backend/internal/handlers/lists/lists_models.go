package lists

import (
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

type rateRequest struct {
	MediaItemId string  `json:"media_item_id"`
	AlbumId     *string `json:"album_id"`
	Status      string  `json:"status"`
	Rating      *int    `json:"rating"`
	// chapters or volumes done, for manga and light novels; left out keeps what's there
	Progress *int `json:"progress"`
}

// updateRequest changes a list entry. Fields left out are left alone, a null rating clears it
type updateRequest struct {
	Status   utils.Optional[string] `json:"status"`
	Rating   utils.Optional[int]    `json:"rating"`
	Progress *int                   `json:"progress"`
}

// ratings run from 1 to 10, nil is no rating
func validRating(rating *int) bool {
	return rating == nil || (*rating >= 1 && *rating <= 10)
}

type UserMediaEntry struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"`
	Rating         *int       `json:"rating"`
	UpdatedAt      time.Time  `json:"updated_at"`
	MediaItemID    *string    `json:"media_item_id"`
	MediaType      string     `json:"media_type"`
	MediaTitle     string     `json:"media_title"`
	CoverImageURL  *string    `json:"cover_image_url"`
	ReleaseDate    *time.Time `json:"release_date"`
	Artist         *string    `json:"artist"`
	AlbumID        *string    `json:"album_id"`
	ActiveSeason   *int       `json:"active_season"`
	SeasonWatched  *int       `json:"season_watched"`
	SeasonTotal    *int       `json:"season_total"`
	ExternalID     *string    `json:"external_id"`
	ExternalSource *string    `json:"external_source"`
	// anime, manga and light novels: episodes, chapters or volumes done (or the count the user
	// gave, when that's further), how many the title has, and whether that total came from the
	// title's source rather than just the files there are
	ProgressCount      *int  `json:"progress_count"`
	ProgressTotal      *int  `json:"progress_total"`
	ProgressTotalKnown *bool `json:"progress_total_known"`
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
