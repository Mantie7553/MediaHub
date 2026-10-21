package curation

import (
	"bytes"
	"encoding/json"
)

/*
optional is a JSON field that can be left out, set to null or set to a value. A plain pointer
can't tell "left out" from null, and null is how a field is cleared
*/
type optional[T any] struct {
	Set   bool
	Value *T
}

func (o *optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// editMediaRequest changes a title's details. Fields left out are left alone
type editMediaRequest struct {
	Title       optional[string]   `json:"title"`
	Description optional[string]   `json:"description"`
	ReleaseDate optional[string]   `json:"release_date"`
	Genres      optional[[]string] `json:"genres"`

	// anime and manga
	Status optional[string] `json:"status"`
	// anime
	Studio optional[string] `json:"studio"`
	// movie
	Director    optional[string] `json:"director"`
	RuntimeMins optional[int]    `json:"runtime_mins"`
	// manga
	TotalChapters optional[int] `json:"total_chapters"`
	// light novel
	Author       optional[string] `json:"author"`
	TotalVolumes optional[int]    `json:"total_volumes"`
	// music track
	Artist      optional[string] `json:"artist"`
	TrackNumber optional[int]    `json:"track_number"`

	// fields to hand back to the library syncs, see package locks
	Unlock []string `json:"unlock"`
}

type editAlbumRequest struct {
	Title       optional[string] `json:"title"`
	Artist      optional[string] `json:"artist"`
	ReleaseDate optional[string] `json:"release_date"`
	Description optional[string] `json:"description"`
}

// editPieceRequest renames or renumbers one episode, chapter or volume. Title null or ""
// goes back to the default name (and for episodes, Sonarr's title)
type editPieceRequest struct {
	Title optional[string] `json:"title"`
	// chapters, written as the number is ("52.5")
	ChapterNumber optional[string] `json:"chapter_number"`
	// volumes
	VolumeNumber optional[int] `json:"volume_number"`
}

type addEpisodeRequest struct {
	UploadID      string `json:"upload_id"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
	Title         string `json:"title"`
}

type addMovieFileRequest struct {
	UploadID string `json:"upload_id"`
}

type addVolumeRequest struct {
	UploadID     string `json:"upload_id"`
	VolumeNumber int    `json:"volume_number"`
	Title        string `json:"title"`
}

type addTrackRequest struct {
	UploadID    string `json:"upload_id"`
	Title       string `json:"title"`
	TrackNumber *int   `json:"track_number"`
}
