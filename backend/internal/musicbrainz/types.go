package musicbrainz

// Artist mirrors the subset of MusicBrainz's artist payload used by MediaHub.
// ID is the MusicBrainz Identifier (MBID), a stable UUID for the entity.
type Artist struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Country string `json:"country"`
	Score   int    `json:"score"`
}

// Release represents an album-level entity. Tracks is populated only by lookup,
// not by search; search results return TrackCount as a hint.
type Release struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Date       string  `json:"date"`
	TrackCount int     `json:"track_count"`
	Tracks     []Track `json:"tracks"`
}

type Track struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	TrackNumber  int    `json:"track_number"`
	LengthMillis int    `json:"length_millis"`
}

// wireArtist mirrors MusicBrainz's artist JSON exactly. Fields happen to match the
// public Artist 1:1 here, but the converter is kept for symmetry with the rest.
type wireArtist struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Country string `json:"country"`
	Score   int    `json:"score"`
}

func (w wireArtist) toDomain() Artist {
	return Artist{
		ID:      w.ID,
		Name:    w.Name,
		Type:    w.Type,
		Country: w.Country,
		Score:   w.Score,
	}
}

// wireRelease mirrors MusicBrainz's release shape: artist credits arrive as an
// ordered chain of fragments, and tracks live under media[] (one entry per disc).
type wireRelease struct {
	ID           string             `json:"id"`
	Title        string             `json:"title"`
	Date         string             `json:"date"`
	TrackCount   int                `json:"track-count"`
	ArtistCredit []wireArtistCredit `json:"artist-credit"`
	Media        []wireMedia        `json:"media"`
}

// wireArtistCredit is one fragment of a credited-artist string. For "Beyoncé feat.
// Jay-Z", MusicBrainz returns three entries: "Beyoncé", " feat. ", "Jay-Z".
type wireArtistCredit struct {
	Name string `json:"name"`
}

// wireMedia is a single physical/logical disc in a release. Its tracks live here,
// not on the release itself, because multi-disc sets distribute tracks across them.
type wireMedia struct {
	Tracks []wireTrack `json:"tracks"`
}

// wireTrack uses Position (always int) rather than Number (string, may be "A1" on vinyl)
// so the public TrackNumber stays cleanly numeric.
type wireTrack struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Title    string `json:"title"`
	Length   int    `json:"length"`
}

func (w wireRelease) toDomain() Release {
	// Concatenate the credit fragments to produce the displayed artist string.
	artist := ""
	for _, c := range w.ArtistCredit {
		artist += c.Name
	}

	// Flatten tracks across every disc so callers see one ordered list.
	tracks := []Track{}
	for _, m := range w.Media {
		for _, t := range m.Tracks {
			tracks = append(tracks, Track{
				ID:           t.ID,
				Title:        t.Title,
				TrackNumber:  t.Position,
				LengthMillis: t.Length,
			})
		}
	}

	return Release{
		ID:         w.ID,
		Title:      w.Title,
		Artist:     artist,
		Date:       w.Date,
		TrackCount: w.TrackCount,
		Tracks:     tracks,
	}
}
