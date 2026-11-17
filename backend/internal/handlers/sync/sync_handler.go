package sync

import (
	"database/sql"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
)

type Handler struct {
	db       *sql.DB
	sonarr   *arr.ArrClient
	radarr   *arr.ArrClient
	mangadex *mangadex.MangaDexClient
	anilist  *anilist.AnilistClient
	runner   runner
	items    keyedLocks // one sync of a series or movie at a time, webhook or full sync
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:       db,
		sonarr:   arr.NewArrClient("SONARR_URL", "SONARR_API_KEY"),
		radarr:   arr.NewArrClient("RADARR_URL", "RADARR_API_KEY"),
		mangadex: mangadex.NewMangaDexClient("MANGADEX_URL"),
		anilist:  anilist.NewAnilistClient(""),
	}
}
