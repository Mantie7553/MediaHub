package ytdlp

// SearchResult mirrors the subset of yt-dlp's --dump-json output used by MediaHub.
// URL is the canonical source URL that should be persisted onto download_requests.
type SearchResult struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Uploader     string `json:"uploader"`
	DurationSecs int    `json:"duration_secs"`
	URL          string `json:"url"`
	Thumbnail    string `json:"thumbnail"`
}
