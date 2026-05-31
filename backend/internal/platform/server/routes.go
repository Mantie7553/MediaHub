package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/handlers/jobs"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/lists"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/media"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/requests"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/search"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/stream"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/sync"
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
	requestsHandler := requests.NewHandler(s.db)
	searchHandler := search.NewHandler(s.db)
	streamHandler := stream.NewHandler(s.db)
	syncHandler := sync.NewHandler(s.db)
	webhooksHandler := webhooks.NewHandler(s.db)

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

	s.router.Post("/webhooks/sonarr", webhooksHandler.SonarrWebhook)

	// Media streaming endpoints
	s.router.Get("/stream/episodes/{id}", streamHandler.StreamEpisode)
	s.router.Get("/stream/segments/{id}/{file}", streamHandler.ServeSegment)

	// Endpoints for all authenticated users
	s.router.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Get("/me", authHandler.Me)

		// Media Handler Endpoints
		r.Get("/media", mediaHandler.GetAll)
		r.Get("/media/{id}", mediaHandler.GetSpecific)
		r.Get("/manga/{id}/chapters/{chapterId}/pages/{pageNum}", mediaHandler.ServePage)
		r.Put("/manga/{id}/chapters/{chapterId}/progress", mediaHandler.MangaProgress)
		r.Get("/media/{id}/episodes", mediaHandler.GetEpisodes)

		// List Handler endpoints
		r.Post("/me/media", listsHandler.Add)
		r.Get("/me/media", listsHandler.GetAll)
		r.Put("/me/media/{id}", listsHandler.Update)
		r.Delete("/me/media/{id}", listsHandler.Delete)
		r.Post("/me/anime/{id}/progress", listsHandler.UpdateProgress)

		// Request Handler endpoints
		r.Get("/requests", requestsHandler.GetAll)
		r.Post("/requests", requestsHandler.Add)

		// Job Handler endpoints
		r.Get("/me/jobs", jobsHandler.GetMine)

		// Search Handler endpoints
		r.Get("/search", searchHandler.Search)
		r.Post("/search/save", searchHandler.Save)

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
	})
}
