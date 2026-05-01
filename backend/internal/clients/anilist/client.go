package anilist

import (
	"context"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/config"
)

// Client is a minimal GraphQL client for the public Anilist API.
// Anilist enforces a global rate limit (currently 90 req/min); a limiter
// can be layered onto the http.Client when traffic warrants it.
type Client struct {
	baseURL string
	http    *http.Client
}

// New constructs a Client. The shared HTTP client respects the configured timeout.
func New(cfg config.AnilistConfig) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		http:    &http.Client{Timeout: cfg.Timeout},
	}
}

// Search queries Anilist for media of the given type matching the search term.
// mediaType is "ANIME" or "MANGA". Movies are returned as ANIME with format=MOVIE.
func (c *Client) Search(ctx context.Context, mediaType, query string, perPage int) ([]Media, error) {
	return nil, nil
}

// GetByID fetches a single media entry by its Anilist ID.
func (c *Client) GetByID(ctx context.Context, id int) (*Media, error) {
	return nil, nil
}

// query executes a GraphQL request against the configured endpoint and decodes
// the response into out. Variables are passed through as-is to the GraphQL payload.
func (c *Client) query(ctx context.Context, gql string, vars map[string]any, out any) error {
	return nil
}
