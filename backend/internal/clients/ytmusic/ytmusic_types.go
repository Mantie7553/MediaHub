package ytmusic

// the kinds of result a search returns
const (
	KindSong  = "song"
	KindVideo = "video"
)

/*
Track is one search result.
Artists: the credited artists in order; Artist joins them for display ("A & B")
Album: the album a song is on (a single's album is usually its own title); empty for videos
Uploader: the channel a video was posted by; for songs, the first artist
Kind: KindSong (audio from YouTube Music's catalogue) or KindVideo
Explicit: YouTube Music marks the track explicit
Official: the label's own upload, official audio for a song or the official music video
Verified: the channel is verified (only known for yt-dlp results)
URL: the canonical watch link that is saved onto the download request
*/
type Track struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Artists      []string `json:"artists"`
	Artist       string   `json:"artist"`
	Album        string   `json:"album"`
	Uploader     string   `json:"uploader"`
	DurationSecs int      `json:"duration_secs"`
	URL          string   `json:"url"`
	Thumbnail    string   `json:"thumbnail"`
	Kind         string   `json:"kind"`
	Explicit     bool     `json:"explicit"`
	Official     bool     `json:"official"`
	Verified     bool     `json:"verified"`
}

// a run is a piece of text in a column, linking to a channel or album when it has a browse id
type run struct {
	Text               string `json:"text"`
	NavigationEndpoint struct {
		BrowseEndpoint struct {
			BrowseID string `json:"browseId"`
		} `json:"browseEndpoint"`
	} `json:"navigationEndpoint"`
}

func (r run) browseID() string { return r.NavigationEndpoint.BrowseEndpoint.BrowseID }

// the part of a musicResponsiveListItemRenderer MediaHub reads
type listItem struct {
	FlexColumns []struct {
		Column struct {
			Text struct {
				Runs []run `json:"runs"`
			} `json:"text"`
		} `json:"musicResponsiveListItemFlexColumnRenderer"`
	} `json:"flexColumns"`
	Badges []struct {
		Badge struct {
			Icon struct {
				IconType string `json:"iconType"`
			} `json:"icon"`
		} `json:"musicInlineBadgeRenderer"`
	} `json:"badges"`
	PlaylistItemData struct {
		VideoID string `json:"videoId"`
	} `json:"playlistItemData"`
	Thumbnail struct {
		Renderer struct {
			Thumbnail struct {
				Thumbnails []struct {
					URL   string `json:"url"`
					Width int    `json:"width"`
				} `json:"thumbnails"`
			} `json:"thumbnail"`
		} `json:"musicThumbnailRenderer"`
	} `json:"thumbnail"`
}
