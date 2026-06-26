package webhooks

type sonarrPayload struct {
	EventType string          `json:"eventType"`
	Series    sonarrSeries    `json:"series"`
	Episodes  []sonarrEpisode `json:"episodes"`
}

type sonarrSeries struct {
	ID int `json:"id"`
}

type sonarrEpisode struct {
	ID            int    `json:"id"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	EpisodeFileID int    `json:"episodeFileId"`
}

type radarrPayload struct {
	EventType string           `json:"eventType"`
	Movie     radarrMovie      `json:"movie"`
	MovieFile radarrMovieFile  `json:"movieFile"`
}

type radarrMovie struct {
	ID int `json:"id"`
}

type radarrMovieFile struct {
	ID int `json:"id"`
}