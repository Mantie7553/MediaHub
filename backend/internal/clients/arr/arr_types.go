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
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Images []struct {
		CoverType string `json:"coverType"`
		RemoteURL string `json:"remoteUrl"`
	} `json:"images"`
	AlternateTitles []struct {
		Title string `json:"title"`
	} `json:"alternateTitles"`
}

func (s SonarrSeries) PosterURL() string {
	for _, img := range s.Images {
		if img.CoverType == "poster" {
			return img.RemoteURL
		}
	}
	return ""
}

type RadarrMovie struct {
	ID              int    `json:"id"`
	Title           string `json:"title"`
	HasFile         bool   `json:"hasFile"`
	DigitalRelease  string `json:"digitalRelease"`
	PhysicalRelease string `json:"physicalRelease"`
	Images          []struct {
		CoverType string `json:"coverType"`
		RemoteURL string `json:"remoteUrl"`
	} `json:"images"`
	MovieFile struct {
		Path string `json:"path"`
	} `json:"movieFile"`
	AlternateTitles []struct {
		Title string `json:"title"`
	} `json:"alternateTitles"`
}

func (m RadarrMovie) PosterURL() string {
	for _, img := range m.Images {
		if img.CoverType == "poster" {
			return img.RemoteURL
		}
	}
	return ""
}

func (m RadarrMovie) ReleaseDate() string {
	if m.DigitalRelease != "" {
		return m.DigitalRelease
	}
	if m.PhysicalRelease != "" {
		return m.PhysicalRelease
	}
	return ""
}
