package clients

import (
	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/musicbrainz"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/plex"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/config"
)

// Set bundles every external-service client so handlers can be wired with a single dependency.
type Set struct {
	Anilist     *anilist.Client
	MusicBrainz *musicbrainz.Client
	Plex        *plex.Client
	YTDLP       *ytdlp.Client
}

// New constructs a Set from an aggregated config. Each client owns its own HTTP transport
// (where applicable) so timeouts and connection reuse are isolated per service.
func New(cfg *config.Config) *Set {
	return &Set{
		Anilist:     anilist.New(cfg.Anilist),
		MusicBrainz: musicbrainz.New(cfg.MusicBrainz),
		Plex:        plex.New(cfg.Plex),
		YTDLP:       ytdlp.New(cfg.YTDLP),
	}
}
