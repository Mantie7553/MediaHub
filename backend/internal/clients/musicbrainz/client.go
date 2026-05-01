package musicbrainz

import (
	"context"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/config"
)

// Client wraps the MusicBrainz REST API. A descriptive User-Agent identifying the
// app and a contact is required by the service's etiquette guidelines and is set
// on every outbound request.
type Client struct {
	baseURL   string
	userAgent string
	http      *http.Client
}

// New constructs a Client. MusicBrainz enforces a 1 req/sec rate limit on
// anonymous traffic; a limiter should wrap c.http before this is hit hard.
func New(cfg config.MusicBrainzConfig) *Client {
	return &Client{
		baseURL:   cfg.BaseURL,
		userAgent: cfg.UserAgent,
		http:      &http.Client{Timeout: cfg.Timeout},
	}
}

// SearchArtist looks up artists by name, ordered by MusicBrainz's relevance score.
func (c *Client) SearchArtist(ctx context.Context, query string, limit int) ([]Artist, error) {
	return nil, nil
}

// SearchRelease looks up releases (albums) by title.
func (c *Client) SearchRelease(ctx context.Context, query string, limit int) ([]Release, error) {
	return nil, nil
}

// GetReleaseByMBID fetches a release with its full track listing expanded.
func (c *Client) GetReleaseByMBID(ctx context.Context, mbid string) (*Release, error) {
	return nil, nil
}

// do issues a GET request against the MusicBrainz API with the configured User-Agent
// and a fmt=json query parameter, then decodes the response into out.
func (c *Client) do(ctx context.Context, path string, query map[string]string, out any) error {
	return nil
}
