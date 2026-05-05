package musicbrainz

import (
	"net/http"
	"os"
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
	// Public endpoint is stable; fall back when the env var is blank.
	if baseURL == "" {
		baseURL = "https://musicbrainz.org/ws/2"
	}
	return &MusicBrainzClient{
		config: MusicBrainzConfig{
			BaseURL:   baseURL,
			UserAgent: os.Getenv(agentEnvKey),
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

/*
	Function:	SearchArtist
	Purpose:	Search MusicBrainz artists by name, ordered by relevance score
	Params:
		- query: free-text artist name
		- limit: number of results to return
*/
func (c *MusicBrainzClient) SearchArtist(query string, limit int) ([]Artist, error) {
	return nil, nil
}

/*
	Function:	SearchRelease
	Purpose:	Search MusicBrainz releases (albums) by title
	Params:
		- query: free-text release title
		- limit: number of results to return
*/
func (c *MusicBrainzClient) SearchRelease(query string, limit int) ([]Release, error) {
	return nil, nil
}

/*
	Function:	GetReleaseByMBID
	Purpose:	Fetch a single release with its full track listing expanded
	Params:
		- mbid: MusicBrainz Identifier for the release
*/
func (c *MusicBrainzClient) GetReleaseByMBID(mbid string) (*Release, error) {
	return nil, nil
}
