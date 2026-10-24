package insights

import "time"

type insightsResponse struct {
	Storage        storageUsage   `json:"storage"`
	Cache          cacheUsage     `json:"cache"`
	TopWatched     []topTitle     `json:"top_watched"`
	TopRead        []topTitle     `json:"top_read"`
	TopPlayed      []topTrack     `json:"top_played"`
	Activity       []activityDay  `json:"activity"`
	StorageHistory []storagePoint `json:"storage_history"`
	Users          userSummary    `json:"users"`
	Jobs           jobSummary     `json:"jobs"`
	GeneratedAt    time.Time      `json:"generated_at"`
}

type folderUsage struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Bytes int64  `json:"bytes"`
	Files int64  `json:"files"`
}

type storageUsage struct {
	Folders []folderUsage `json:"folders"`
	// transcodes and unfinished uploads, cleared after 48 hours
	Temp folderUsage `json:"temp"`
	// the volume MEDIA_ROOT is on; nil when it can't be read
	DiskTotal *int64    `json:"disk_total"`
	DiskFree  *int64    `json:"disk_free"`
	ScannedAt time.Time `json:"scanned_at"`
}

type cacheUsage struct {
	Enabled bool  `json:"enabled"`
	Used    int64 `json:"used"`
	Max     int64 `json:"max"`
}

type topTitle struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Users int    `json:"users"`
	// episodes or chapters/volumes touched in the window, across users
	Items int `json:"items"`
	// hours watched, from saved positions; 0 for reading
	Hours float64 `json:"hours"`
}

type userActivity struct {
	ID       string     `json:"id"`
	Username string     `json:"username"`
	Role     string     `json:"role"`
	LastSeen *time.Time `json:"last_seen"`
	// titles watched or read in the window
	RecentTitles int `json:"recent_titles"`
}

type userSummary struct {
	Total    int            `json:"total"`
	Active7  int            `json:"active_7d"`
	Active30 int            `json:"active_30d"`
	List     []userActivity `json:"list"`
}

type failedJob struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Handler  *string    `json:"handler"`
	Error    *string    `json:"error"`
	FailedAt *time.Time `json:"failed_at"`
}

type jobSummary struct {
	// job counts by status over the window
	Counts          map[string]int `json:"counts"`
	PendingRequests int            `json:"pending_requests"`
	RecentFailures  []failedJob    `json:"recent_failures"`
}

type topTrack struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Artist  string  `json:"artist"`
	AlbumID *string `json:"album_id"`
	Users   int     `json:"users"`
	Plays   int     `json:"plays"`
}

// one day of the activity chart
type activityDay struct {
	Day   string `json:"day"` // YYYY-MM-DD
	Users int    `json:"users"`
	Plays int    `json:"plays"`
}

// one day's storage snapshot
type storagePoint struct {
	Day          string `json:"day"` // YYYY-MM-DD
	LibraryBytes int64  `json:"library_bytes"`
	DiskUsed     *int64 `json:"disk_used"`
	DiskTotal    *int64 `json:"disk_total"`
}
