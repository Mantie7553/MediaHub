package anilist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

/*
BaseURL: GraphQL endpoint for Anilist
*/
type AnilistConfig struct {
	BaseURL string
}

/*
config: BaseURL for the Anilist API
http: pointer to the http client
*/
type AnilistClient struct {
	config AnilistConfig
	http   *http.Client
}

/*
Function:	NewAnilistClient
Purpose:	Connect with the Anilist GraphQL API
Params:
  - urlEnvKey: environment variable key holding the Anilist URL
*/
func NewAnilistClient(urlEnvKey string) *AnilistClient {
	baseURL := os.Getenv(urlEnvKey)
	// Public endpoint is stable; fall back when the .env var is blank rather than failing every call.
	if baseURL == "" {
		baseURL = "https://graphql.anilist.co"
	}
	return &AnilistClient{
		config: AnilistConfig{
			BaseURL: baseURL,
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// searchQuery is the GraphQL document used by Search. Declared at package scope so the
// raw multi-line string stays out of the method body.
const searchQuery = `query ($search: String, $type: MediaType, $perPage: Int) {
  Page(perPage: $perPage) {
    media(search: $search, type: $type) {
      id
      type
      format
      status
      title { romaji english native }
      description
      episodes
      chapters
      volumes
      coverImage { large medium }
      genres
      studios { nodes { name } }
      startDate { year month day }
    }
  }
}`

// getByIDQuery is the GraphQL document used by GetByID. Same fields as searchQuery,
// but at the Media root instead of nested inside Page.
const getByIDQuery = `query ($id: Int) {
  Media(id: $id) {
    id
    type
    format
    status
    title { romaji english native }
    description
    episodes
    chapters
    volumes
    coverImage { large medium }
    genres
    studios { nodes { name } }
    startDate { year month day }
  }
}`

/*
Function:	Search
Purpose:	Search Anilist for media of a given type matching the query
Params:
  - mediaType: "ANIME" or "MANGA". Empty string searches across both.
  - query: free-text search term
  - perPage: number of results to return; defaults to 10 when zero or negative
*/
func (c *AnilistClient) Search(mediaType, query string, perPage int) ([]Media, error) {
	if perPage <= 0 {
		perPage = 10
	}

	vars := map[string]any{
		"search":  query,
		"perPage": perPage,
	}
	// Omit type from the variables when empty so Anilist doesn't reject the value as invalid enum.
	if mediaType != "" {
		vars["type"] = mediaType
	}

	var env struct {
		Page struct {
			Media []wireMedia `json:"media"`
		} `json:"Page"`
	}
	if err := c.query(searchQuery, vars, &env); err != nil {
		return nil, err
	}

	// Map each wire result into the public Media shape used everywhere else in the app.
	out := make([]Media, 0, len(env.Page.Media))
	for _, m := range env.Page.Media {
		out = append(out, m.toDomain())
	}
	return out, nil
}

/*
Function:	GetByID
Purpose:	Fetch a single Anilist media entry by its numeric ID
Params:
  - id: Anilist's numeric ID for the media entry
*/
func (c *AnilistClient) GetByID(id int) (*Media, error) {
	vars := map[string]any{"id": id}

	var env struct {
		Media *wireMedia `json:"Media"`
	}
	if err := c.query(getByIDQuery, vars, &env); err != nil {
		return nil, err
	}

	if env.Media == nil {
		return nil, nil
	}
	domain := env.Media.toDomain()
	return &domain, nil
}

// graphQLRequest is the JSON body sent to the GraphQL endpoint on every call.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLError mirrors a single entry in the errors array Anilist returns when
// a query fails server-side. The full struct has more fields; we only need the message.
type graphQLError struct {
	Message string `json:"message"`
}

// graphQLResponse wraps the data and errors fields of any GraphQL response.
// Data is held as RawMessage so the helper can defer decoding until after error checks pass.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

// query executes a GraphQL request and decodes the data field into out.
// Both transport-level errors (5xx, network failures) and GraphQL-level errors
// (the errors array, which arrives with HTTP 200) are surfaced to the caller.
func (c *AnilistClient) query(gql string, vars map[string]any, out any) error {
	body, err := json.Marshal(graphQLRequest{Query: gql, Variables: vars})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.config.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("anilist returned %d", resp.StatusCode)
	}

	var envelope graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}

	// GraphQL servers return 200 even on validation failures; the errors array tells the truth.
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("anilist graphql error: %s", envelope.Errors[0].Message)
	}

	if out == nil || len(envelope.Data) == 0 {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}
