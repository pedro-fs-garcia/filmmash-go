package web

import (
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"filmmash/internal/middleware"
	"filmmash/internal/vote"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func InitRouter(
	logger *slog.Logger,
	metrics *metrics.Metrics,
	filmService *film.Service,
	duelService *duel.Service,
	registerVoteUC *vote.RegisterVoteUC,
) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer(logger))
	router.Use(middleware.RequestID)
	router.Use(middleware.ResponseLogger(logger))
	router.Use(metrics.HttpMetrics.MetricsMiddleware)

	handler := NewHandler(
		logger.With("component", "Handler"),
		filmService,
		duelService,
		registerVoteUC,
	)

	router.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))

	router.Route("/ui", func(r chi.Router) {
		r.Get("/", handler.DuelHandler)
		r.Get("/films", handler.FilmsListHandler)
		r.Get("/films/search", handler.SearchFilmsHandler)
		r.Get("/film/{id}", handler.FilmHandler)
		r.Post("/duel/{duel_id}/result", handler.DuelResultHandler)
	})

	return router
}
