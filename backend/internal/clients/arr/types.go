package arr

type SonarrEpisode struct {
	ID            int    `json:"id"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	EpisodeFileID int    `json:"episodeFileId"`
	HasFile       bool   `json:"hasFile"`
}

type SonarrSeries struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type RadarrMovie struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	HasFile   bool   `json:"hasFile"`
	MovieFile struct {
		Path string `json:"path"`
	} `json:"movieFile"`
}
