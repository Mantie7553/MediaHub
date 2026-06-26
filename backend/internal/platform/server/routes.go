package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/handlers/jobs"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/lists"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/media"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/music"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/requests"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/search"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/stream"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/sync"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/users"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/webhooks"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type Server struct {
	router *chi.Mux
	db     *sql.DB
}

func New(db *sql.DB) *Server {
	s := &Server{
		router: chi.NewRouter(),
		db:     db,
	}
	s.routes()
	return s
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) routes() {
	authHandler := auth.NewHandler(s.db)
	jobsHandler := jobs.NewHandler(s.db)
	listsHandler := lists.NewHandler(s.db)
	mediaHandler := media.NewHandler(s.db)
	musicHandler := music.NewHandler(s.db)
	requestsHandler := requests.NewHandler(s.db)
	searchHandler := search.NewHandler(s.db)
	streamHandler := stream.NewHandler(s.db)
	syncHandler := sync.NewHandler(s.db)
	webhooksHandler := webhooks.NewHandler(s.db)
	usersHandler := users.NewHandler(s.db)

	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}))

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	s.router.Post("/auth/register", authHandler.Register)
	s.router.Post("/auth/login", authHandler.Login)
	s.router.Delete("/auth/logout", authHandler.Logout)
	s.router.Post("/auth/refresh", authHandler.Refresh)

	s.router.Get("/light-novels/{id}/volumes/{volumeId}/images/{imageName}", mediaHandler.ServeVolumeImage)

	s.router.Post("/webhooks/sonarr", webhooksHandler.SonarrWebhook)
	s.router.Post("/webhooks/radarr", webhooksHandler.RadarrWebhook)

	// Media streaming endpoints
	s.router.Get("/stream/media/{type}/{id}", streamHandler.StreamMedia)
	s.router.Get("/stream/segments/{type}/{id}/{file}", streamHandler.ServeSegment)
	s.router.Get("/stream/music/{id}", streamHandler.ServeTrack)
	s.router.Get("/stream/music/{id}/cover", streamHandler.ServeCover)

	// Endpoints for all authenticated users
	s.router.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Get("/me", authHandler.Me)

		// Media Handler Endpoints
		r.Get("/media", mediaHandler.GetAll)
		r.Get("/media/{id}", mediaHandler.GetSpecific)
		r.Put("/episodes/{id}/progress", mediaHandler.UpdateEpisodeProgress)
		r.Get("/manga/{id}/chapters/{chapterId}/pages/{pageNum}", mediaHandler.ServePage)
		r.Put("/manga/{id}/chapters/{chapterId}/progress", mediaHandler.MangaProgress)
		r.Get("/media/{id}/episodes", mediaHandler.GetEpisodes)
		r.Put("/manga/chapters/{chapterId}/read", mediaHandler.MarkChapterRead)
		r.Put("/manga/{id}/read", mediaHandler.MarkMangaRead)
		r.Put("/light-novels/volumes/{volumeId}/read", mediaHandler.MarkVolumeRead)
		r.Put("/light-novels/volumes/{volumeId}/progress", mediaHandler.UpdateLightNovelProgress)
		r.Put("/light-novels/{id}/read", mediaHandler.MarkLightNovelRead)

		//Music Handler Endpoints
		r.Get("/albums", musicHandler.GetAlbums)
		r.Get("/albums/{id}", musicHandler.GetAlbum)
		r.Get("/music/recommended", musicHandler.GetRecommended)

		// List Handler endpoints
		r.Get("/light-novels/{id}/volumes/{volumeId}/content", mediaHandler.ServeVolume)
		r.Post("/me/media", listsHandler.Add)
		r.Get("/me/media", listsHandler.GetAll)
		r.Put("/me/media/{id}", listsHandler.Update)
		r.Delete("/me/media/{id}", listsHandler.Delete)
		r.Put("/episodes/{id}/watched", listsHandler.UpdateProgress)
		r.Put("/anime/{id}/watched", listsHandler.MarkShowWatched)
		r.Put("/anime/{id}/seasons/{seasonNumber}/watched", listsHandler.MarkSeasonWatched)

		// Request Handler endpoints
		r.Get("/requests", requestsHandler.GetAll)
		r.Post("/requests", requestsHandler.Add)

		// Job Handler endpoints
		r.Get("/me/jobs", jobsHandler.GetMine)

		// Search Handler endpoints
		r.Get("/search", searchHandler.Search)
		r.Post("/search/save", searchHandler.Save)
		r.Get("/music/yt-search", searchHandler.YTSearch)

	})

	// Endpoints for admin users
	s.router.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Use(auth.RequireAdmin)
		r.Post("/media", mediaHandler.Upload)
		// Request Handler endpoints
		r.Get("/requests/all", requestsHandler.GetAllAdmin)
		r.Put("/requests/{id}/approve", requestsHandler.Approve)
		r.Put("/requests/{id}/reject", requestsHandler.Reject)

		// Job Handler endpoints
		r.Get("/admin/jobs", jobsHandler.GetAll)
		r.Post("/admin/jobs", jobsHandler.Create)

		// Sync Handler endpoints
		r.Post("/admin/sync/sonarr", syncHandler.SyncSonar)
		r.Post("/admin/sync/radarr", syncHandler.SyncRadarr)
		r.Post("/admin/sync/manga", syncHandler.SyncManga)
		r.Post("/admin/sync/light-novels", syncHandler.SyncLightNovel)
		r.Post("/admin/sync/music", syncHandler.SyncMusic)

		// User Handler endpoints
		r.Get("/admin/users", usersHandler.GetAll)
		r.Post("/admin/users", usersHandler.Create)
		r.Put("/admin/users/{id}", usersHandler.Update)
		r.Delete("/admin/users/{id}", usersHandler.Delete)
	})
}
