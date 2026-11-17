package arr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// ErrNotFound: Sonarr / Radarr has no such series, movie or file
var ErrNotFound = errors.New("not found")

/*
Function:	getJSON
Purpose:	GET an API path and decode the JSON reply into out. A 404 is ErrNotFound
Params:
  - path: the API path, e.g. "/api/v3/series/12"
  - out: where to decode the reply
*/
func (c *ArrClient) getJSON(path string, out any) error {
	req, err := http.NewRequest("GET", c.config.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

/*
Function:	GetSeries
Purpose:	one Sonarr series
Params:
  - seriesID: the Sonarr series id
*/
func (c *ArrClient) GetSeries(seriesID int) (SonarrSeries, error) {
	var s SonarrSeries
	err := c.getJSON("/api/v3/series/"+strconv.Itoa(seriesID), &s)
	return s, err
}

/*
Function:	GetMovie
Purpose:	one Radarr movie
Params:
  - movieID: the Radarr movie id
*/
func (c *ArrClient) GetMovie(movieID int) (RadarrMovie, error) {
	var m RadarrMovie
	err := c.getJSON("/api/v3/movie/"+strconv.Itoa(movieID), &m)
	return m, err
}

/*
Function:	GetEpisodeFiles
Purpose:	every file of a Sonarr series in one call, by episode file id to its path
Params:
  - seriesID: the Sonarr series id
*/
func (c *ArrClient) GetEpisodeFiles(seriesID int) (map[int]string, error) {
	var files []struct {
		ID   int    `json:"id"`
		Path string `json:"path"`
	}
	if err := c.getJSON("/api/v3/episodefile?seriesId="+strconv.Itoa(seriesID), &files); err != nil {
		return nil, err
	}
	out := make(map[int]string, len(files))
	for _, f := range files {
		out[f.ID] = f.Path
	}
	return out, nil
}
