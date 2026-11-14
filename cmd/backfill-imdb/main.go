package main

import (
	"context"
	"filmmash/internal/config"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"filmmash/internal/tmdb"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"
)

func main() {
	delay := flag.Duration("delay", 100*time.Millisecond, "minimum interval between TMDB calls")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	cfg := config.Load()
	dsn := &url.URL{Scheme: "postgres", User: url.UserPassword(cfg.PostgresUser, cfg.PostgresPassword), Host: cfg.PostgresHost + ":" + cfg.PostgresPort, Path: cfg.PgDbName}
	q := dsn.Query()
	q.Set("sslmode", cfg.PostgresSSLMode)
	dsn.RawQuery = q.Encode()
	pool, err := database.Pool(ctx, dsn.String())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	queries := dbgen.New(pool)
	ids, err := queries.FilmsMissingImdb(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := tmdb.NewClient("https://api.themoviedb.org/3", cfg.TmdbApitoken)
	failures := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		m, err := client.GetFilm(id)
		if err != nil {
			log.Printf("film %d: %v", id, err)
			failures++
		} else if m.ImdbId == "" {
			log.Printf("film %d has no IMDB id", id)
		} else if err = queries.UpdateFilmImdb(ctx, dbgen.UpdateFilmImdbParams{ID: id, ImdbID: &m.ImdbId}); err != nil {
			log.Printf("film %d: %v", id, err)
			failures++
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(*delay):
		}
	}
	if failures > 0 {
		log.Fatal(fmt.Sprintf("%d films failed; rerun to retry", failures))
	}
}
