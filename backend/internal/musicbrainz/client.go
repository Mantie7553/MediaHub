package musicbrainz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

/*
BaseURL: REST endpoint for MusicBrainz
UserAgent: identifying string required by MusicBrainz etiquette
*/
type MusicBrainzConfig struct {
	BaseURL   string
	UserAgent string
}

/*
config: BaseURL and UserAgent for the MusicBrainz API
http: pointer to the http client
*/
type MusicBrainzClient struct {
	config MusicBrainzConfig
	http   *http.Client
}

/*
Function:	NewMusicBrainzClient
Purpose:	Connect with the MusicBrainz REST API
Params:
  - urlEnvKey: environment variable key holding the MusicBrainz URL
  - agentEnvKey: environment variable key holding the User-Agent string
*/
func NewMusicBrainzClient(urlEnvKey string, agentEnvKey string) *MusicBrainzClient {
	baseURL := os.Getenv(urlEnvKey)
	if baseURL == "" {
		baseURL = "https://musicbrainz.org/ws/2"
	}
	userAgent := os.Getenv(agentEnvKey)
	// MusicBrainz rejects blank User-Agent strings; provide an identifiable fallback.
	if userAgent == "" {
		userAgent = "MediaHub/0.1 ( https://github.com/Mantie7553/MediaHub )"
	}
	return &MusicBrainzClient{
		config: MusicBrainzConfig{
			BaseURL:   baseURL,
			UserAgent: userAgent,
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

/*
Function:	SearchArtist
Purpose:	Search MusicBrainz artists by name, ordered by relevance score
Params:
  - query: free-text artist name
  - limit: number of results to return; defaults to 10 when zero or negative
*/
func (c *MusicBrainzClient) SearchArtist(query string, limit int) ([]Artist, error) {
	if limit <= 0 {
		limit = 10
	}

	var env struct {
		Artists []wireArtist `json:"artists"`
	}
	err := c.do("/artist", map[string]string{
		"query": query,
		"limit": strconv.Itoa(limit),
	}, &env)
	if err != nil {
		return nil, err
	}

	out := make([]Artist, 0, len(env.Artists))
	for _, a := range env.Artists {
		out = append(out, a.toDomain())
	}
	return out, nil
}

/*
Function:	SearchRelease
Purpose:	Search MusicBrainz releases (albums) by title
Params:
  - query: free-text release title
  - limit: number of results to return; defaults to 10 when zero or negative
*/
func (c *MusicBrainzClient) SearchRelease(query string, limit int) ([]Release, error) {
	if limit <= 0 {
		limit = 10
	}

	var env struct {
		Releases []wireRelease `json:"releases"`
	}
	err := c.do("/release", map[string]string{
		"query": query,
		"limit": strconv.Itoa(limit),
	}, &env)
	if err != nil {
		return nil, err
	}

	out := make([]Release, 0, len(env.Releases))
	for _, r := range env.Releases {
		out = append(out, r.toDomain())
	}
	return out, nil
}

/*
Function:	GetReleaseByMBID
Purpose:	Fetch a single release with its full track listing expanded
Params:
  - mbid: MusicBrainz Identifier for the release
*/
func (c *MusicBrainzClient) GetReleaseByMBID(mbid string) (*Release, error) {
	var w wireRelease
	// inc=recordings populates media[].tracks[]; artist-credits supplies the credited artist string.
	err := c.do("/release/"+mbid, map[string]string{
		"inc": "recordings artist-credits",
	}, &w)
	if err != nil {
		return nil, err
	}
	domain := w.toDomain()
	return &domain, nil
}

// do issues a GET against the MusicBrainz API with the configured User-Agent and
// the fmt=json query parameter, then decodes the response into out.
func (c *MusicBrainzClient) do(path string, query map[string]string, out any) error {
	u, err := url.Parse(c.config.BaseURL + path)
	if err != nil {
		return err
	}

	q := u.Query()
	// MusicBrainz returns XML by default; fmt=json is required for JSON.
	q.Set("fmt", "json")
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("musicbrainz returned %d", resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
