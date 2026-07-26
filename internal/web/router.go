package web

import (
	"filmmash/internal/admin"
	"filmmash/internal/auth"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"filmmash/internal/middleware"
	"filmmash/internal/vote"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	logger *slog.Logger,
	HTTPMetrics *metrics.HttpMetrics,
	authService *auth.Service,
	authHandler *auth.Handler,
	filmHandler *film.Handler,
	duelHandler *duel.Handler,
	voteHandler *vote.Handler,
	adminHandler *admin.Handler,
) *chi.Mux {
	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	router.Group(func(r chi.Router) {
		r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
		r.Use(middleware.RequestID)
		r.Use(middleware.ResponseLogger(logger))
		r.Use(HTTPMetrics.MetricsMiddleware)
		r.Use(middleware.Recoverer(logger))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/ui", http.StatusFound)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Get("/login", authHandler.LoginHandler)
			r.Get("/callback", authHandler.CallbackHandler)
			r.Get("/logout", authHandler.LogoutHandler)
		})

		r.Route("/ui", func(r chi.Router) {
			r.Get("/", duelHandler.DuelHandler)
			r.Get("/films", filmHandler.FilmsListHandler)
			r.Get("/films/search", filmHandler.SearchFilmsHandler)
			r.Get("/film/{id}", filmHandler.FilmHandler)
			r.Get("/film/votes/{film_id}", voteHandler.ListFilmVotes)
			r.Get("/my_votes", authService.SessionMiddleware(voteHandler.MyVotesHandler))
			r.Post(
				"/duel/{duel_id}/result",
				authService.SessionMiddleware(voteHandler.DuelResultHandler),
			)
			r.Get("/duel/from/{film_id}", duelHandler.DuelFromFilm)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(authService.SessionCtx)
			r.Use(auth.RequiresRole("admin"))
			r.Get("/", adminHandler.ListUsersPaginatedHandler)
			r.Get("/users", adminHandler.ListUsersPaginatedHandler)
			r.Get("/films/search", adminHandler.SearchFilmOnTmdbHandler)
			r.Post("/films/insert_from_tmdb/{id}", adminHandler.AddTmdbFilmHandler)
		})

		r.Route("/user", func(r chi.Router) {
			r.Use(authService.SessionCtx)
			r.Get("/console", authHandler.UserConsoleHandler)
		})
	})

	return router
}
