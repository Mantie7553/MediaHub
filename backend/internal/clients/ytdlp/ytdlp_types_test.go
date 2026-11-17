package ytdlp

import "testing"

func TestToDomainFallsBackToChannelAndURL(t *testing.T) {
	got := wireSearchResult{
		ID:       "H58vbez_m4E",
		Title:    "Kendrick Lamar - Not Like Us",
		Channel:  "Kendrick Lamar",
		Verified: true,
		Duration: 355.4,
		URL:      "https://www.youtube.com/watch?v=H58vbez_m4E",
	}.toDomain()

	if got.Uploader != "Kendrick Lamar" {
		t.Errorf("uploader = %q, want the channel", got.Uploader)
	}
	if got.URL != "https://www.youtube.com/watch?v=H58vbez_m4E" {
		t.Errorf("url = %q", got.URL)
	}
	if !got.Verified || got.DurationSecs != 355 {
		t.Errorf("got %+v", got)
	}
	if got.Thumbnail == "" {
		t.Error("expected a fallback thumbnail")
	}
}

func TestToDomainPrefersUploader(t *testing.T) {
	got := wireSearchResult{Uploader: "Uploader", Channel: "Channel", WebpageURL: "a", URL: "b"}.toDomain()
	if got.Uploader != "Uploader" || got.URL != "a" {
		t.Errorf("got %+v", got)
	}
}
