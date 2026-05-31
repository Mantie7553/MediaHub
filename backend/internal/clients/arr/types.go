package arr

type SonarrEpisode struct {
	ID            int    `json:"id"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	EpisodeFileID int    `json:"episodeFileId"`
	HasFile       bool   `json:"hasFile"`
}
