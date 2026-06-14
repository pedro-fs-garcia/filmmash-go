package web

import (
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/middleware"
	"filmmash/internal/vote"

	"github.com/go-chi/chi/v5"
)

func InitRouter(
	filmService *film.Service,
	duelService *duel.Service,
	registerVoteUC *vote.RegisterVoteUC,
) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use()

	handler := NewHandler(
		filmService,
		duelService,
		registerVoteUC,
	)

	router.Route("/ui", func(r chi.Router) {
		r.Get("/", handler.DuelHandler)
		r.Get("/film/{id}", handler.FilmHandler)
		r.Post("/duel/{duel_id}/result", handler.DuelResultHandler)
	})

	return router
}
