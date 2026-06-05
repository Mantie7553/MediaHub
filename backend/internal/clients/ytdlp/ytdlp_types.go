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

// wireSearchResult mirrors yt-dlp's --dump-json output for a single search hit.
// Duration arrives as a JSON number (sometimes int, sometimes float) so we decode
// into float64 and round on conversion to keep parsing tolerant.
type wireSearchResult struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Uploader   string  `json:"uploader"`
	Duration   float64 `json:"duration"`
	WebpageURL string  `json:"webpage_url"`
	Thumbnail  string  `json:"thumbnail"`
}

func (w wireSearchResult) toDomain() SearchResult {
	thumb := w.Thumbnail
	if thumb == "" {
		thumb = "https://img.youtube.com/vi/" + w.ID + "/mqdefault.jpg"
	}
	return SearchResult{
		ID:           w.ID,
		Title:        w.Title,
		Uploader:     w.Uploader,
		DurationSecs: int(w.Duration),
		URL:          w.WebpageURL,
		Thumbnail:    thumb,
	}
}
