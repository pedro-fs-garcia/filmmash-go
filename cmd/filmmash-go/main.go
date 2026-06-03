package main

import (
	"context"
	"encoding/json"
	"filmmash/internal/config"
	"filmmash/internal/database"
	"filmmash/internal/tmdb"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
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

	service := Service{pool: pool}
	handler := AppHandler{service: &service}

	router := chi.NewRouter()
	router.Route("/ui", func(r chi.Router) {
		r.Get("/film/{id}", handler.FilmHandler)
	})

	httpServer := &http.Server{Addr: "localhost:8000", Handler: router}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", httpServer.Addr, err)
	}

	log.Printf("server listening on http://%s\n", httpServer.Addr)

	if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func temp(w http.ResponseWriter, r *http.Request) {
	data, err := tmdb.NewTmdbClient().GetTopRated(50)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
