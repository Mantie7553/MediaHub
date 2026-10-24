package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/handlers/collections"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/curation"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/deletion"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/insights"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/invites"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/jobs"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/lists"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/manga"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/media"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/music"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/requests"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/search"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/stream"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/sync"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/users"
	"github.com/Mantie7553/MediaHub/backend/internal/handlers/webhooks"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/archive"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/uploads"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type Server struct {
	router   *chi.Mux
	db       *sql.DB
	sync     *sync.Handler
	search   *search.Handler
	insights *insights.Handler
}

func New(db *sql.DB) *Server {
	s := &Server{
		router: chi.NewRouter(),
		db:     db,
	}
	s.routes()
	return s
}

/*
Function:	StartBackgroundJobs
Purpose:	start work that runs on its own schedule (the nightly library sync, the daily
storage snapshot), and fetch Discover's lists so the first visit doesn't wait for them
*/
func (s *Server) StartBackgroundJobs() {
	s.sync.StartSchedule()
	s.search.WarmDiscovery()
	s.insights.StartDailySnapshots()
}

func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
		// a client that sends its headers slowly can't hold connections open forever.
		// There is no overall write timeout: streams and long downloads stay open
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
	}
	return srv.ListenAndServe()
}

func (s *Server) routes() {
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		log.Fatal("CORS_ORIGIN not set")
	}
	if os.Getenv("MEDIA_SIGNING_SECRET") == "" {
		log.Fatal("MEDIA_SIGNING_SECRET not set")
	}
	if os.Getenv("MEDIA_SIGNING_SECRET") == os.Getenv("JWT_SECRET") {
		log.Fatal("MEDIA_SIGNING_SECRET must be different from JWT_SECRET")
	}
	for _, key := range []string{"JWT_SECRET", "MEDIA_SIGNING_SECRET"} {
		if len(os.Getenv(key)) < 32 {
			logger.Warn("%s is shorter than 32 characters, tokens signed with it are easier to forge; the install script generates a long one", key)
		}
	}
	authHandler := auth.NewHandler(s.db)
	curationHandler := curation.NewHandler(s.db)
	uploadsStore := uploads.Default()
	invitesHandler := invites.NewHandler(s.db)
	jobsHandler := jobs.NewHandler(s.db)
	listsHandler := lists.NewHandler(s.db)
	collectionsHandler := collections.NewHandler(s.db)
	mangaHandler := manga.NewHandler(s.db)
	mediaHandler := media.NewHandler(s.db)
	deletionHandler := deletion.NewHandler(s.db)
	musicHandler := music.NewHandler(s.db)
	requestsHandler := requests.NewHandler(s.db)
	searchHandler := search.NewHandler(s.db)
	s.search = searchHandler
	streamHandler := stream.NewHandler(s.db)
	syncHandler := sync.NewHandler(s.db)
	s.sync = syncHandler
	webhooksHandler := webhooks.NewHandler(s.db)
	usersHandler := users.NewHandler(s.db)
	insightsHandler := insights.NewHandler(s.db)
	s.insights = insightsHandler

	s.router.Use(secureHeaders)
	s.router.Use(limitBody)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{corsOrigin},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}))

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	s.router.Route("/api", func(r chi.Router) {
		// the session cookie's endpoints, limited per IP more loosely than sign in since every
		// open tab refreshes its access token
		r.Group(func(r chi.Router) {
			r.Use(auth.RateLimitSessions)
			r.Delete("/auth/logout", authHandler.Logout)
			r.Post("/auth/refresh", authHandler.Refresh)
		})

		r.Get("/proxy/cover", mediaHandler.ProxyCover)

		// Sonarr/Radarr webhooks, authorized by the shared secret header
		r.Group(func(r chi.Router) {
			r.Use(webhooks.RequireSecret)
			r.Post("/webhooks/sonarr", webhooksHandler.SonarrWebhook)
			r.Post("/webhooks/radarr", webhooksHandler.RadarrWebhook)
		})

		// Byte-serving endpoints loaded by <video>/<audio>/<img>, authorized by a
		// short lived media token minted from the authenticated endpoints below
		r.With(auth.RequireMediaToken(func(r *http.Request) auth.MediaScope {
			return auth.MediaScope{MediaType: chi.URLParam(r, "type"), MediaID: chi.URLParam(r, "id")}
		})).Get("/stream/segments/{type}/{id}/{variant}/{file}", streamHandler.ServeSegment)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireMediaToken(func(r *http.Request) auth.MediaScope {
				return auth.MediaScope{MediaType: "music", MediaID: chi.URLParam(r, "id")}
			}))
			r.Get("/stream/music/{id}", streamHandler.ServeTrack)
			r.Get("/stream/music/{id}/cover", streamHandler.ServeCover)
		})

		// covers admins uploaded for a title
		r.With(auth.RequireMediaToken(func(r *http.Request) auth.MediaScope {
			return auth.MediaScope{MediaType: "cover", MediaID: chi.URLParam(r, "id")}
		})).Get("/media/{id}/cover", curationHandler.ServeCover)

		r.With(auth.RequireMediaToken(func(r *http.Request) auth.MediaScope {
			return auth.MediaScope{
				MediaType: "light_novel",
				MediaID:   chi.URLParam(r, "id"),
				VolumeID:  chi.URLParam(r, "volumeId"),
			}
		})).Get("/light-novels/{id}/volumes/{volumeId}/images/{imageName}", mediaHandler.ServeVolumeImage)

		// Endpoints that are rate limited
		r.Group(func(r chi.Router) {
			r.Use(auth.RateLimit)

			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
		})

		// Endpoints for all authenticated users
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware)
			r.Get("/me", authHandler.Me)
			// account changes check the current password, so they are rate limited too
			r.With(auth.RateLimit).Put("/me", authHandler.UpdateAccount)
			// under /auth so the browser sends the session cookie (Path=/api/auth) and the session
			// making the change stays signed in. /me/password is the old address, for tabs
			// opened before an update; without the cookie it signs out every session
			r.With(auth.RateLimit).Put("/auth/password", authHandler.ChangePassword)
			r.With(auth.RateLimit).Put("/me/password", authHandler.ChangePassword)
			r.Post("/auth/logout-all", authHandler.LogoutAll)
			r.Get("/me/preferences", usersHandler.GetPreferences)
			r.Put("/me/preferences", usersHandler.UpdatePreferences)

			// Media streaming endpoints, these mint the media tokens for the byte routes
			r.Get("/stream/media/{type}/{id}", streamHandler.StreamMedia)
			r.Get("/stream/music/{id}/url", streamHandler.SignTrackURL)

			// Media Handler Endpoints
			r.Get("/media", mediaHandler.GetAll)
			r.Get("/media/{id}", mediaHandler.GetSpecific)
			r.Put("/episodes/{id}/progress", mediaHandler.UpdateEpisodeProgress)
			r.Put("/movies/{id}/progress", mediaHandler.UpdateMovieProgress)
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
			r.Get("/music/artists", musicHandler.GetArtists)
			r.Post("/music/{id}/plays", musicHandler.RecordPlay)

			// List Handler endpoints
			r.Get("/light-novels/{id}/volumes/{volumeId}/content", mediaHandler.ServeVolume)
			r.Post("/me/media", listsHandler.Add)
			r.Get("/me/media", listsHandler.GetAll)
			r.Get("/me/continue", listsHandler.GetContinue)
			// playlists and collections; the computed playlists answer to /collections/listened and /unlistened
			r.Get("/me/collections", collectionsHandler.List)
			r.Post("/me/collections", collectionsHandler.Create)
			r.Get("/collections/{id}", collectionsHandler.Get)
			r.Put("/collections/{id}", collectionsHandler.Update)
			r.Delete("/collections/{id}", collectionsHandler.Delete)
			r.Post("/collections/{id}/items", collectionsHandler.AddItem)
			r.Delete("/collections/{id}/items/{entryId}", collectionsHandler.RemoveItem)
			r.Put("/collections/{id}/items/{entryId}/position", collectionsHandler.MoveItem)
			// featuring: users submit theirs, everyone sees the featured copies and can save one as
			// their own
			r.Get("/featured", collectionsHandler.ListFeatured)
			r.Post("/collections/{id}/submission", collectionsHandler.Submit)
			r.Delete("/collections/{id}/submission", collectionsHandler.Withdraw)
			r.Post("/collections/{id}/copy", collectionsHandler.SaveCopy)
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
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware)
			r.Use(auth.RequireAdmin)
			r.Post("/media", mediaHandler.Upload)
			// permanent deletion: files, Sonarr/Radarr entry and every user's lists and progress
			r.Delete("/media/{id}", deletionHandler.DeleteMedia)
			r.Delete("/albums/{id}", deletionHandler.DeleteAlbum)
			r.Post("/admin/albums/{id}/cover", musicHandler.UploadCover)
			// manga chapters: upload a file, or fetch what MangaDex has that the library doesn't
			r.Post("/admin/manga/{id}/chapters", mangaHandler.UploadChapter)
			r.Post("/admin/manga/{id}/check", mangaHandler.CheckForChapters)

			// curating the library by hand: details, covers, and the pieces of each title.
			// Edited fields are locked so the library syncs keep them
			r.Put("/admin/media/{id}", curationHandler.EditMedia)
			r.Post("/admin/media/{id}/cover", curationHandler.UploadCover)
			r.Put("/admin/albums/{id}", curationHandler.EditAlbum)
			r.Put("/admin/episodes/{id}", curationHandler.EditEpisode)
			r.Put("/admin/chapters/{id}", curationHandler.EditChapter)
			r.Put("/admin/volumes/{id}", curationHandler.EditVolume)
			r.Delete("/admin/episodes/{id}", deletionHandler.DeleteEpisode)
			r.Delete("/admin/chapters/{id}", deletionHandler.DeleteChapter)
			r.Delete("/admin/volumes/{id}", deletionHandler.DeleteVolume)
			// large files arrive in chunks (Cloudflare allows 100 MB a request), then are
			// added to a title by naming the finished upload
			r.Post("/admin/uploads", uploadsStore.Start)
			r.Put("/admin/uploads/{id}", uploadsStore.Chunk)
			r.Delete("/admin/uploads/{id}", uploadsStore.Cancel)
			r.Post("/admin/anime/{id}/episodes", curationHandler.AddEpisode)
			r.Post("/admin/movies/{id}/file", curationHandler.AddMovieFile)
			r.Post("/admin/light-novels/{id}/volumes", curationHandler.AddVolume)
			r.Post("/admin/albums/{id}/tracks", curationHandler.AddTrack)
			// Request Handler endpoints
			r.Get("/requests/all", requestsHandler.GetAllAdmin)
			r.Put("/requests/{id}/approve", requestsHandler.Approve)
			r.Put("/requests/{id}/reject", requestsHandler.Reject)
			// hide requests and jobs from the Downloads lists, or bring them back
			r.Post("/requests/archive", archive.Handler(s.db, "download_requests"))

			// Job Handler endpoints
			r.Get("/admin/jobs", jobsHandler.GetAll)
			r.Post("/admin/jobs", jobsHandler.Create)
			r.Post("/admin/jobs/archive", archive.Handler(s.db, "download_jobs"))

			// Sync Handler endpoints
			r.Post("/admin/sync", syncHandler.StartSync)
			r.Get("/admin/sync", syncHandler.GetSyncStatus)
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

			// the Insights page: disk use, what people watch and read, users and downloads
			r.Get("/admin/insights", insightsHandler.Get)

			// reviewing what users submit to be featured, and featuring an admin's own
			r.Get("/admin/collection-submissions", collectionsHandler.ListSubmissions)
			r.Post("/admin/collection-submissions/{id}/approve", collectionsHandler.Approve)
			r.Post("/admin/collection-submissions/{id}/reject", collectionsHandler.Reject)
			r.Post("/admin/collections/{id}/feature", collectionsHandler.Feature)

			// Invite Handler endpoints
			r.Get("/admin/invites", invitesHandler.GetAll)
			r.Post("/admin/invites", invitesHandler.Create)
			r.Delete("/admin/invites/{id}", invitesHandler.Revoke)
		})
	})
}
