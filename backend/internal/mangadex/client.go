package mangadex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

/*
BaseURL: Endpoint for MangaDex
*/
type MangaDexConfig struct {
	BaseURL string
}

/*
config: BaseURL for the MangaDex API
http: pointer to the http client
*/
type MangaDexClient struct {
	config MangaDexConfig
	http   *http.Client
}

/*
Function:	NewMangaDexClient
Purpose:	Connect with the MangaDex API
Params:
  - urlEnvKey: environment variable key holding the MangaDex URL
*/
func NewMangaDexClient(urlEnvKey string) *MangaDexClient {
	baseURL := os.Getenv(urlEnvKey)
	// Public endpoint is stable; fall back when the .env var is blank rather than failing every call.
	if baseURL == "" {
		baseURL = "https://api.mangadex.org"
	}
	return &MangaDexClient{
		config: MangaDexConfig{
			BaseURL: baseURL,
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

/*
Function:	Search
Purpose:	Search MangaDex for manga matching the query
Params:
  - query: the query we want to run when looking for any manga
*/
func (c *MangaDexClient) Search(query string) ([]Manga, error) {
	params := url.Values{}
	params.Set("title", query)
	params.Set("limit", "20")
	params.Add("includes[]", "cover_art")

	req, err := http.NewRequest("GET", c.config.BaseURL+"/manga?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mangadex returned %d", resp.StatusCode)
	}

	var result struct {
		Data []Manga `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
