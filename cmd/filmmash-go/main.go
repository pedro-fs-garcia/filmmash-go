package main

import (
	"encoding/json"
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
	router := chi.NewRouter()
	registerHandlers(router)
	RegisterUiHandlers(router)

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

func registerHandlers(router *chi.Mux) {
	router.Get("/", HomeHandler)
	router.Get("/health", HandleHealth)
	router.Get("/tmdb", temp)

	router.Route("/api", func(r chi.Router) {

	})

	router.Get("/{id}", IdHandler)
}

func RegisterUiHandlers(router *chi.Mux) {
	router.Route("/ui", func(r chi.Router) {
		r.Get("/", HomeHandler)
		r.Get("/film/{id}", FilmHandler)
	})
}
