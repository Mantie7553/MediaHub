package musicbrainz

import (
	"net/http"
	"os"
	"time"
)

/*
BaseURL: URL of the local Plex Media Server
Token: X-Plex-Token used to authenticate every request
*/
type PlexConfig struct {
	BaseURL string
	Token   string
}

/*
config: BaseURL and Token for the Plex API
http: pointer to the http client
*/
type PlexClient struct {
	config PlexConfig
	http   *http.Client
}

/*
Function:	NewPlexClient
Purpose:	Connect with a local Plex Media Server using a server-scoped token
Params:
  - urlEnvKey: environment variable key holding the Plex URL
  - tokenEnvKey: environment variable key holding the Plex token
*/
func NewPlexClient(urlEnvKey string, tokenEnvKey string) *PlexClient {
	return &PlexClient{
		config: PlexConfig{
			BaseURL: os.Getenv(urlEnvKey),
			Token:   os.Getenv(tokenEnvKey),
		},
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

/*
Function:	GetLibraries
Purpose:	List every library section (movies, shows, music, etc.) on the server
*/
func (c *PlexClient) GetLibraries() ([]Library, error) {
	return nil, nil
}

/*
Function:	GetLibraryItems
Purpose:	List metadata items inside a single library section
Params:
  - sectionID: id of the library section to enumerate
*/
func (c *PlexClient) GetLibraryItems(sectionID string) ([]Item, error) {
	return nil, nil
}

/*
Function:	GetItem
Purpose:	Fetch a single metadata item by its ratingKey
Params:
  - ratingKey: Plex's stable identifier for the item
*/
func (c *PlexClient) GetItem(ratingKey string) (*Item, error) {
	return nil, nil
}

/*
Function:	StreamURL
Purpose:	Build an authenticated direct-stream URL the browser can hit
Params:
  - ratingKey: Plex's stable identifier for the item
*/
func (c *PlexClient) StreamURL(ratingKey string) string {
	return ""
}
