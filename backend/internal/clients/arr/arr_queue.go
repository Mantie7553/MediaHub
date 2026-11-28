package arr

import (
	"strconv"
	"time"
)

// QueueItem is one download in Sonarr's / Radarr's queue: a release it grabbed and is waiting
// on, downloading or importing
type QueueItem struct {
	ID       int `json:"id"`
	SeriesID int `json:"seriesId"`
	MovieID  int `json:"movieId"`
	// the release's name
	Title  string `json:"title"`
	Status string `json:"status"`
	// ok, warning or error: Sonarr / Radarr's verdict on it, e.g. a stalled torrent is a warning
	TrackedDownloadStatus string `json:"trackedDownloadStatus"`
	// downloading, importPending, importBlocked, importing, imported, failedPending, failed, ignored
	TrackedDownloadState string `json:"trackedDownloadState"`
	StatusMessages       []struct {
		Title    string   `json:"title"`
		Messages []string `json:"messages"`
	} `json:"statusMessages"`
	ErrorMessage string `json:"errorMessage"`
	// when it was grabbed, zero when Sonarr / Radarr doesn't say
	Added time.Time `json:"added"`
}

// queuePageSize is how many queue items are asked for at once
const queuePageSize = 200

/*
Function:	Configured
Purpose:	whether Sonarr / Radarr's URL is set, so it can be called at all
*/
func (c *ArrClient) Configured() bool {
	return c.config.BaseURL != ""
}

/*
Function:	GetQueue
Purpose:	every item in Sonarr's / Radarr's download queue, page by page
*/
func (c *ArrClient) GetQueue() ([]QueueItem, error) {
	var items []QueueItem
	for page := 1; ; page++ {
		var reply struct {
			TotalRecords int         `json:"totalRecords"`
			Records      []QueueItem `json:"records"`
		}
		path := "/api/v3/queue?page=" + strconv.Itoa(page) + "&pageSize=" + strconv.Itoa(queuePageSize)
		if err := c.getJSON(path, &reply); err != nil {
			return nil, err
		}
		items = append(items, reply.Records...)
		if len(reply.Records) == 0 || len(items) >= reply.TotalRecords {
			return items, nil
		}
	}
}
