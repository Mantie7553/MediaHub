package webhooks

// the fields MediaHub reads of Sonarr's webhook events (Download, Rename, EpisodeFileDelete,
// SeriesDelete, Test, ...)
type sonarrPayload struct {
	EventType    string          `json:"eventType"`
	Series       sonarrSeries    `json:"series"`
	Episodes     []sonarrEpisode `json:"episodes"`
	DeleteReason string          `json:"deleteReason"`
}

type sonarrSeries struct {
	ID int `json:"id"`
}

type sonarrEpisode struct {
	ID int `json:"id"`
}

// the fields MediaHub reads of Radarr's webhook events (Download, Rename, MovieFileDelete,
// MovieDelete, Test, ...)
type radarrPayload struct {
	EventType    string      `json:"eventType"`
	Movie        radarrMovie `json:"movie"`
	DeleteReason string      `json:"deleteReason"`
}

type radarrMovie struct {
	ID int `json:"id"`
}
