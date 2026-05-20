package plex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type PlexConfig struct {
	BaseURL   string
	Token     string
	MachineID string
}

type PlexClient struct {
	config PlexConfig
	http   *http.Client
}

func NewPlexClient(urlEnvKey, tokenEnvKey string) *PlexClient {
	return &PlexClient{
		config: PlexConfig{
			BaseURL:   os.Getenv(urlEnvKey),
			Token:     os.Getenv(tokenEnvKey),
			MachineID: os.Getenv("PLEX_MACHINE_ID"),
		},
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *PlexClient) GetLibraries() ([]Directory, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/library/sections", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", c.config.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result PlexResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.MediaContainer.Directory, nil
}

func (c *PlexClient) GetPartKey(ratingKey string) (string, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/library/metadata/"+ratingKey, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", c.config.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result PlexResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	md := result.MediaContainer.Metadata
	if len(md) == 0 || len(md[0].Media) == 0 || len(md[0].Media[0].Part) == 0 {
		return "", fmt.Errorf("no part found for rating key %s", ratingKey)
	}

	return md[0].Media[0].Part[0].Key, nil
}

func (c *PlexClient) StreamURL(ratingKey string) string {
	return fmt.Sprintf(
		"%s/web/index.html#!/server/%s/details?key=%%2Flibrary%%2Fmetadata%%2F%s",
		c.config.BaseURL, c.config.MachineID, ratingKey,
	)
}
