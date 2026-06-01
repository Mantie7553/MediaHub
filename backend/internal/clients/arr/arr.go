package arr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

/*
BaseURL: URL to reach Sonarr or Radarr
APIKey: Key to access Sonarr or Radarr API
*/
type ArrConfig struct {
	BaseURL string
	APIKey  string
}

/*
config: URL and APIKey for Sonarr / Radarr
http: pointer to the http client
*/
type ArrClient struct {
	config ArrConfig
	http   *http.Client
}

/*
Function:	NewArrClient
Purpose:	Connect with Sonarr or Radarr
Params:
  - urlEnvKey: environment variables URL key name
  - apiEnvKey: environment variables API_KEY key name
*/
func NewArrClient(urlEnvKey string, apiEnvKey string) *ArrClient {
	return &ArrClient{
		config: ArrConfig{
			BaseURL: os.Getenv(urlEnvKey),
			APIKey:  os.Getenv(apiEnvKey),
		},
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

/*
Function:	AddSeries
Purpose:	Add a series for tracking to the Sonarr client
Params:
  - tvdbID: the id of the series to track
  - qualityProfileID: the id of the quality profile to use while searching
  - rootFolderPath: the location we will send content to once it has been downloaded
*/
func (c *ArrClient) AddSeries(tvdbID int, qualityProfileID int, rootFolderPath string) (int, error) {
	// Initialize the request body we are going to send
	body, err := json.Marshal(map[string]any{
		"tvdbId":           tvdbID,
		"qualityProfileId": qualityProfileID,
		"rootFolderPath":   rootFolderPath,
		"monitored":        true,
		"seasonFolder":     true,
	})
	if err != nil {
		return 0, err
	}

	// create the request
	req, err := http.NewRequest("POST", c.config.BaseURL+"/api/v3/series", bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.config.APIKey)

	// Make the request
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// check that it worked or not
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("sonarr returned %d", resp.StatusCode)
	}

	// get the Sonarr series ID or 0
	var result struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.ID, nil
}

func (c *ArrClient) GetAllSeries() ([]SonarrSeries, error) {
	var result []SonarrSeries
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/series", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Sonarr returned %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *ArrClient) GetEpisodes(seriesID int) ([]SonarrEpisode, error) {
	var result []SonarrEpisode
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/episode?seriesId="+strconv.Itoa(seriesID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Sonarr returned %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *ArrClient) GetEpisodeFilePath(episodeFileID int) (string, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/episodefile/"+strconv.Itoa(episodeFileID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Sonarr returned %d", resp.StatusCode)
	}
	var result struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Path, nil
}

/*
Function:	AddMovie
Purpose:	Add a movie for tracking to the Radarr client
Params:
  - tmdbID: the id of the movie to track
  - qualityProfileID: the id of the quality profile to use while searching
  - rootFolderPath: the location we will send content to once it has been downloaded
*/
func (c *ArrClient) AddMovie(tmdbID int, qualityProfileID int, rootFolderPath string) (int, error) {
	// Initialize the request body we are going to send
	body, err := json.Marshal(map[string]any{
		"tmdbId":           tmdbID,
		"qualityProfileId": qualityProfileID,
		"rootFolderPath":   rootFolderPath,
		"monitored":        true,
	})
	if err != nil {
		return 0, err
	}

	// create the request
	req, err := http.NewRequest("POST", c.config.BaseURL+"/api/v3/movie", bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.config.APIKey)

	// Make the request
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// check that it worked or not
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("radarr returned %d", resp.StatusCode)
	}

	// get the Radarr movie ID or 0
	var result struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.ID, nil
}

func (c *ArrClient) GetAllMovies() ([]RadarrMovie, error) {
	var result []RadarrMovie
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/movie", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Radarr returned %d", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Radarr returned %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
