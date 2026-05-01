package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/lists"
	"github.com/Mantie7553/MediaHub/backend/internal/media"
	"github.com/Mantie7553/MediaHub/backend/internal/requests"
	"github.com/go-chi/chi/v5"
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
	mediaHandler := media.NewHandler(s.db)
	listsHandler := lists.NewHandler(s.db)
	requestsHandler := requests.NewHandler(s.db)

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	s.router.Post("/auth/register", authHandler.Register)
	s.router.Post("/auth/login", authHandler.Login)

	// Endpoints for all authenticated users
	s.router.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Get("/me", authHandler.Me)
		r.Get("/media", mediaHandler.GetAll)
		r.Get("/media/{id}", mediaHandler.GetSpecific)

		r.Post("/me/media", listsHandler.Add)
    	r.Get("/me/media", listsHandler.GetAll)
    	r.Put("/me/media/{id}", listsHandler.Update)
    	r.Delete("/me/media/{id}", listsHandler.Delete)
    	r.Post("/me/anime/{id}/progress", listsHandler.UpdateProgress)

		r.Get("/requests", requestsHandler.GetAll)
		r.Post("/requests", requestsHandler.Add)

	})

	// Endpoints for admin users
	s.router.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Use(auth.RequireAdmin)
		r.Post("/media", mediaHandler.Upload)
		r.Get("/admin/test", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "admin only")
		})

		r.Get("/requests/all", requestsHandler.GetAllAdmin)
		r.Put("/requests/{id}/approve", requestsHandler.Approve)
		r.Put("/requests/{id}/reject", requestsHandler.Reject)
	})
}
