package arr

import "time"

type SonarrEpisode struct {
	ID            int    `json:"id"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	EpisodeFileID int    `json:"episodeFileId"`
	HasFile       bool   `json:"hasFile"`
	// when it first aired, empty when Sonarr doesn't know yet
	AirDateUtc string `json:"airDateUtc"`
}

/*
Function:	Listed
Purpose:	whether the library lists an episode: every one with a file, and every aired one of a
regular season, so a series shows all its seasons (and counts them in progress) before they're
downloaded. Specials without a file are left out, Sonarr lists many nobody watches, and so are
episodes still to air
Params:
  - now: the time now
*/
func (e SonarrEpisode) Listed(now time.Time) bool {
	if e.HasFile {
		return true
	}
	if e.SeasonNumber == 0 || e.AirDateUtc == "" {
		return false
	}
	aired, err := time.Parse(time.RFC3339, e.AirDateUtc)
	return err == nil && !aired.After(now)
}

type SonarrSeries struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	TvdbID   int      `json:"tvdbId"`
	Overview string   `json:"overview"`
	Genres   []string `json:"genres"`
	Year     int      `json:"year"`
	Images   []struct {
		CoverType string `json:"coverType"`
		RemoteURL string `json:"remoteUrl"`
	} `json:"images"`
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
	ID              int      `json:"id"`
	Title           string   `json:"title"`
	HasFile         bool     `json:"hasFile"`
	DigitalRelease  string   `json:"digitalRelease"`
	PhysicalRelease string   `json:"physicalRelease"`
	TmdbID          int      `json:"tmdbId"`
	Overview        string   `json:"overview"`
	Genres          []string `json:"genres"`
	Year            int      `json:"year"`
	Images          []struct {
		CoverType string `json:"coverType"`
		RemoteURL string `json:"remoteUrl"`
	} `json:"images"`
	MovieFile struct {
		Path string `json:"path"`
	} `json:"movieFile"`
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
