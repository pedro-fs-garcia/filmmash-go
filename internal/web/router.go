package web

import (
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"filmmash/internal/middleware"
	"filmmash/internal/vote"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func InitRouter(
	metrics *metrics.Metrics,
	filmService *film.Service,
	duelService *duel.Service,
	registerVoteUC *vote.RegisterVoteUC,
) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(metrics.HttpMetrics.MetricsMiddleware)

	handler := NewHandler(
		filmService,
		duelService,
		registerVoteUC,
	)

	router.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))

	router.Route("/ui", func(r chi.Router) {
		r.Get("/", handler.DuelHandler)
		r.Get("/film/{id}", handler.FilmHandler)
		r.Post("/duel/{duel_id}/result", handler.DuelResultHandler)
	})

	return router
}
