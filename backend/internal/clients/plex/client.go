package plex

import (
	"context"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/config"
)

// Client talks to a local Plex Media Server using a server-scoped X-Plex-Token.
// The token is attached to every request as a query parameter so direct stream
// URLs returned by StreamURL are usable from a browser <audio>/<video> element.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(cfg config.PlexConfig) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		token:   cfg.Token,
		http:    &http.Client{Timeout: cfg.Timeout},
	}
}

// GetLibraries returns all library sections (movies, shows, music, etc.) on the server.
func (c *Client) GetLibraries(ctx context.Context) ([]Library, error) {
	return nil, nil
}

// GetLibraryItems returns metadata items in a single library section.
func (c *Client) GetLibraryItems(ctx context.Context, sectionID string) ([]Item, error) {
	return nil, nil
}

// GetItem fetches a single metadata item by its ratingKey.
func (c *Client) GetItem(ctx context.Context, ratingKey string) (*Item, error) {
	return nil, nil
}

// StreamURL builds an authenticated direct-stream URL for the given ratingKey.
// Used by the audio module so the browser <audio> element streams from Plex
// directly, without proxying bytes through the Go backend.
func (c *Client) StreamURL(ratingKey string) string {
	return ""
}

// do issues a JSON-formatted request against the Plex API with the auth token attached
// and decodes the response into out.
func (c *Client) do(ctx context.Context, path string, query map[string]string, out any) error {
	return nil
}
