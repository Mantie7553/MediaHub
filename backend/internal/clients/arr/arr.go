package arr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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
Function:	QualityProfileID
Purpose:	Look up a Sonarr / Radarr quality profile id by its name (case insensitive)
Params:
  - name: the profile name, e.g. "Anime"
*/
func (c *ArrClient) QualityProfileID(name string) (int, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/qualityprofile", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("quality profile lookup returned %d", resp.StatusCode)
	}
	var profiles []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return 0, err
	}
	for _, p := range profiles {
		if strings.EqualFold(p.Name, name) {
			return p.ID, nil
		}
	}
	return 0, fmt.Errorf("no quality profile named %q", name)
}

/*
Function:	AddSeries
Purpose:	Add a series for tracking to the Sonarr client
Params:
  - tvdbID: the id of the series to track
  - qualityProfileID: the id of the quality profile to use while searching
  - rootFolderPath: the location we will send content to once it has been downloaded
*/
func (c *ArrClient) AddSeries(tvdbID int, qualityProfileID int, rootFolderPath string, title string) (int, error) {
	// Initialize the request body we are going to send
	body, err := json.Marshal(map[string]any{
		"tvdbId":           tvdbID,
		"title":            title,
		"qualityProfileId": qualityProfileID,
		// anime type searches by absolute episode number, which is how most anime releases are named
		"seriesType":     "anime",
		"rootFolderPath": rootFolderPath,
		"monitored":      true,
		"seasonFolder":   true,
		"addOptions": map[string]any{
			"searchForMissingEpisodes": true,
			"monitor":                  "all",
		},
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
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("sonarr returned %d: %s", resp.StatusCode, string(body))
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

func (c *ArrClient) LookupSeries(title string) (tvdbID int, sonarrID int, err error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/series/lookup?term="+url.QueryEscape(title), nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("sonarr lookup returned %d", resp.StatusCode)
	}

	var results []struct {
		ID     int `json:"id"`
		TvdbID int `json:"tvdbId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, err
	}
	if len(results) == 0 {
		return 0, 0, fmt.Errorf("no results found for %s", title)
	}
	return results[0].TvdbID, results[0].ID, nil
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
		return nil, fmt.Errorf("Radarr request failed: %w", err)
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

func (c *ArrClient) GetMovieFilePath(movieFileID int) (string, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/moviefile/"+strconv.Itoa(movieFileID), nil)
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
		return "", fmt.Errorf("radarr returned %d", resp.StatusCode)
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
Function:	DeleteSeries
Purpose:	Remove a series from Sonarr along with its files, so it is neither monitored nor
            downloaded again. A series that is already gone counts as deleted
Params:
  - seriesID: the Sonarr series id
*/
func (c *ArrClient) DeleteSeries(seriesID int) error {
	return c.delete(fmt.Sprintf("/api/v3/series/%d?deleteFiles=true&addImportListExclusion=false", seriesID))
}

/*
Function:	DeleteMovie
Purpose:	Remove a movie from Radarr along with its files, so it is neither monitored nor
            downloaded again. A movie that is already gone counts as deleted
Params:
  - movieID: the Radarr movie id
*/
func (c *ArrClient) DeleteMovie(movieID int) error {
	return c.delete(fmt.Sprintf("/api/v3/movie/%d?deleteFiles=true&addImportExclusion=false", movieID))
}

/*
Function:	UnmonitorEpisode
Purpose:	tell Sonarr to stop watching for an episode, so removing its file doesn't make Sonarr
            download it again
Params:
  - episodeID: the Sonarr episode id
*/
func (c *ArrClient) UnmonitorEpisode(episodeID int) error {
	body, err := json.Marshal(map[string]any{"episodeIds": []int{episodeID}, "monitored": false})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", c.config.BaseURL+"/api/v3/episode/monitor", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

/*
Function:	EpisodeFileID
Purpose:	the id of the file Sonarr has for an episode, 0 when it has none (or no longer knows
            the episode)
Params:
  - episodeID: the Sonarr episode id
*/
func (c *ArrClient) EpisodeFileID(episodeID int) (int, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/api/v3/episode/"+strconv.Itoa(episodeID), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Sonarr returned %d", resp.StatusCode)
	}
	var ep SonarrEpisode
	if err := json.NewDecoder(resp.Body).Decode(&ep); err != nil {
		return 0, err
	}
	if !ep.HasFile {
		return 0, nil
	}
	return ep.EpisodeFileID, nil
}

/*
Function:	DeleteEpisodeFile
Purpose:	have Sonarr delete one episode's file from disk. A file that is already gone counts
            as deleted
Params:
  - episodeFileID: the Sonarr episode file id
*/
func (c *ArrClient) DeleteEpisodeFile(episodeFileID int) error {
	return c.delete(fmt.Sprintf("/api/v3/episodefile/%d", episodeFileID))
}

/*
Function:	delete
Purpose:	send a DELETE to Sonarr / Radarr, treating 404 (already removed) as success
Params:
  - path: the API path with its query
*/
func (c *ArrClient) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.config.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode/100 == 2 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
