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

func (c *MangaDexClient) fetch(params url.Values) ([]Manga, error) {
    params.Add("includes[]", "cover_art")
    params.Add("includes[]", "tag")
    params.Add("excludedTags[]", "b13b2a48-c720-44a9-9c77-39c9979373fb") // doujinshi

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

func (c *MangaDexClient) Discovery(limit int) (*DiscoveryResult, error) {
    if limit <= 0 {
        limit = 20
    }
    limitStr := fmt.Sprintf("%d", limit)

    trendingParams := url.Values{}
    trendingParams.Set("limit", limitStr)
    trendingParams.Add("order[followedCount]", "desc")

    popularParams := url.Values{}
    popularParams.Set("limit", limitStr)
    popularParams.Add("order[rating]", "desc")

    latestParams := url.Values{}
    latestParams.Set("limit", limitStr)
    latestParams.Add("order[latestUploadedChapter]", "desc")

    trending, err := c.fetch(trendingParams)
    if err != nil {
        return nil, fmt.Errorf("trending: %w", err)
    }

    popular, err := c.fetch(popularParams)
    if err != nil {
        return nil, fmt.Errorf("popular: %w", err)
    }

    latest, err := c.fetch(latestParams)
    if err != nil {
        return nil, fmt.Errorf("latest: %w", err)
    }

    return &DiscoveryResult{
        Trending: trending,
        Popular:  popular,
        Latest:   latest,
    }, nil
}

/*
Function:	Search
Purpose:	Search MangaDex for manga matching the query
Params:
  - query: the query we want to run when looking for any manga
*/
func (c *MangaDexClient) Search(query string) ([]Manga, error) {
	params := url.Values{}
	params.Add("excludedTags[]", "b13b2a48-c720-44a9-9c77-39c9979373fb")
	params.Set("title", query)
	params.Set("limit", "20")
	params.Add("includes[]", "cover_art")
	params.Add("includes[]", "tag")

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

/*
Function:	Trending
Purpose:	Search MangaDex for Trending Manga
*/
func (c *MangaDexClient) Trending() ([]Manga, error) {
	params := url.Values{}
	params.Add("excludedTags[]", "b13b2a48-c720-44a9-9c77-39c9979373fb")
	params.Add("order[followedCount]", "desc")
	params.Set("limit", "20")
	params.Add("includes[]", "cover_art")
	params.Add("includes[]", "tag")

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

/*
Function:	GetByID
Purpose:	Fetch a single MangaDex entry by its external (MangaDex) id
Params:
  - id: MangaDex's ID for the manga entry
*/
func (c *MangaDexClient) GetByID(id string) (*Manga, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/manga/"+id+"?includes[]=tag", nil)

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
		Data Manga `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
