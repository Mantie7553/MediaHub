package search

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytmusic"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// a music search: the label's songs on YouTube Music, then videos (music videos, live
// versions, other people's uploads)
type musicSearchResponse struct {
	Songs  []ytmusic.Track `json:"songs"`
	Videos []ytmusic.Track `json:"videos"`
}

/*
Function:	YTSearch
Purpose:	Search YouTube Music for songs and videos to download
Params:
  - w: http response writer
  - r: http request with query params ?q= and an optional ?limit= (5 to 25 songs)
*/
func (h *Handler) YTSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	l := r.URL.Query().Get("limit")
	if q == "" {
		utils.Error(w, http.StatusBadRequest, "q is required")
		return
	}

	limit, err := strconv.Atoi(l)
	if err != nil || (limit < 5 || limit > 25) {
		limit = 15
	}

	results, err := h.musicSearches.Get(searchKey(strconv.Itoa(limit), q), func() (musicSearchResponse, error) {
		return h.searchMusic(q, limit)
	})
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, results)
}

/*
Function:	searchMusic
Purpose:	search YouTube Music's songs and videos side by side, up to limit songs and half as
many videos. YouTube Music's search isn't a published API, so when it gives nothing at all the
search falls back to yt-dlp's YouTube search, whose results are listed as videos
*/
func (h *Handler) searchMusic(q string, limit int) (musicSearchResponse, error) {
	var (
		wg                  sync.WaitGroup
		songs, videos       []ytmusic.Track
		songsErr, videosErr error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		songs, songsErr = h.ytm.Search(q, ytmusic.FilterSongs, limit)
	}()
	go func() {
		defer wg.Done()
		videos, videosErr = h.ytm.Search(q, ytmusic.FilterVideos, (limit+1)/2)
	}()
	wg.Wait()

	if songsErr != nil {
		logger.Warn("youtube music song search for %q failed: %v", q, songsErr)
	}
	if videosErr != nil {
		logger.Warn("youtube music video search for %q failed: %v", q, videosErr)
	}
	if len(songs) > 0 || len(videos) > 0 {
		return musicSearchResponse{Songs: nonNil(songs), Videos: nonNil(videos)}, nil
	}

	logger.Warn("youtube music search for %q gave nothing, falling back to yt-dlp", q)
	found, err := h.yt.SearchMusic(q, limit)
	if err != nil {
		return musicSearchResponse{}, err
	}
	fallback := make([]ytmusic.Track, 0, len(found))
	for _, f := range found {
		fallback = append(fallback, fromYtdlp(f))
	}
	return musicSearchResponse{Songs: []ytmusic.Track{}, Videos: fallback}, nil
}

// fromYtdlp lists a yt-dlp search result as a video
func fromYtdlp(r ytdlp.SearchResult) ytmusic.Track {
	artists := []string{}
	if r.Uploader != "" {
		artists = append(artists, r.Uploader)
	}
	return ytmusic.Track{
		ID:           r.ID,
		Title:        r.Title,
		Artists:      artists,
		Artist:       r.Uploader,
		Uploader:     r.Uploader,
		DurationSecs: r.DurationSecs,
		URL:          r.URL,
		Thumbnail:    r.Thumbnail,
		Kind:         ytmusic.KindVideo,
		Verified:     r.Verified,
	}
}

// nonNil keeps an empty list from being sent as null
func nonNil(t []ytmusic.Track) []ytmusic.Track {
	if t == nil {
		return []ytmusic.Track{}
	}
	return t
}
