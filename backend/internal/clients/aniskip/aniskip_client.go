package aniskip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

/*
BaseURL: AniSkip's API
*/
type AniSkipConfig struct {
	BaseURL string
}

/*
config: BaseURL for the AniSkip API
http: pointer to the http client
*/
type AniSkipClient struct {
	config AniSkipConfig
	http   *http.Client
}

// Segment is a part of an episode to skip, in seconds. Type is op, ed or recap
type Segment struct {
	Type  string  `json:"type"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type skipTimesResponse struct {
	Found   bool `json:"found"`
	Results []struct {
		Interval struct {
			StartTime float64 `json:"startTime"`
			EndTime   float64 `json:"endTime"`
		} `json:"interval"`
		SkipType string `json:"skipType"`
	} `json:"results"`
}

// AniSkip's skip types and what the player calls them; a mixed opening is one over the episode
var skipTypes = map[string]string{
	"op":       "op",
	"mixed-op": "op",
	"ed":       "ed",
	"mixed-ed": "ed",
	"recap":    "recap",
}

/*
Function:	NewAniSkipClient
Purpose:	Connect with the AniSkip API, community timestamps of anime openings and endings
*/
func NewAniSkipClient() *AniSkipClient {
	return &AniSkipClient{
		config: AniSkipConfig{BaseURL: "https://api.aniskip.com"},
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

/*
Function:	SkipTimes
Purpose:	the openings, endings and recaps AniSkip knows for an episode, none when it has none
Params:
  - malID: the MyAnimeList id of the show (season)
  - episode: the episode number within it
  - length: the length of the file in seconds, AniSkip leaves out times for other cuts
*/
func (c *AniSkipClient) SkipTimes(malID, episode int, length float64) ([]Segment, error) {
	query := url.Values{}
	for t := range skipTypes {
		query.Add("types[]", t)
	}
	query.Set("episodeLength", strconv.FormatFloat(length, 'f', 3, 64))
	endpoint := fmt.Sprintf("%s/v2/skip-times/%d/%d?%s", c.config.BaseURL, malID, episode, query.Encode())

	resp, err := c.http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 404 is how it says it has no times for the episode
	if resp.StatusCode == http.StatusNotFound {
		return []Segment{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aniskip returned %d", resp.StatusCode)
	}

	var parsed skipTimesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	segments := []Segment{}
	for _, r := range parsed.Results {
		kind, ok := skipTypes[r.SkipType]
		if !ok || r.Interval.EndTime <= r.Interval.StartTime {
			continue
		}
		segments = append(segments, Segment{Type: kind, Start: r.Interval.StartTime, End: r.Interval.EndTime})
	}
	return segments, nil
}
