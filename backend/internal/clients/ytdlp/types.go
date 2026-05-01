package ytdlp

// SearchResult mirrors the subset of yt-dlp's --dump-json output used by MediaHub.
// URL is the canonical source URL that should be persisted onto download_requests.
type SearchResult struct {
	ID           string
	Title        string
	Uploader     string
	DurationSecs int
	URL          string
	Thumbnail    string
}
