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
	router.Use(middleware.RequestID)
	router.Use(middleware.ResponseLogger(logger))
	router.Use(HTTPMetrics.MetricsMiddleware)
	router.Use(middleware.Recoverer(logger))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui", http.StatusFound)
	})

	router.Route("/auth", func(r chi.Router) {
		r.Get("/login", authHandler.LoginHandler)
		r.Get("/callback", authHandler.CallbackHandler)
		r.Get("/logout", authHandler.LogoutHandler)
	})

	router.Route("/ui", func(r chi.Router) {
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
	})

	router.Route("/admin", func(r chi.Router) {
		r.Use(authService.SessionCtx)
		r.Use(auth.RequiresRole("admin"))
		r.Get("/", adminHandler.ListUsersPaginatedHandler)
		r.Get("/users", adminHandler.ListUsersPaginatedHandler)
	})

	router.Route("/user", func(r chi.Router) {
		r.Use(authService.SessionCtx)
		r.Get("/console", authHandler.UserConsoleHandler)
	})

	return router
}
