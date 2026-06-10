package main

import (
	"context"
	"filmmash/internal/config"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/vote"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	cfg := config.Load()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PgDbName,
	)

	pool, err := database.Pool(ctx, dsn)
	if err != nil {
		panic(err)
	}

	filmService := film.NewService(pool)
	duelService := duel.NewService(pool, filmService)
	voteService := vote.NewService(pool, filmService, duelService)

	handler := NewAppHandler(
		filmService,
		duelService,
		voteService,
	)

	router := chi.NewRouter()
	router.Route("/ui", func(r chi.Router) {
		r.Get("/", handler.DuelHandler)
		r.Get("/film/{id}", handler.FilmHandler)
		r.Post("/duel/{duel_id}/result", handler.DuelResultHandler)
	})

	router.Route("/api", func(r chi.Router) {
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", httpServer.Addr, err)
	}

	log.Printf("server listening on port %s\n", httpServer.Addr)

	if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
