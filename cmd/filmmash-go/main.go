package main

import (
	"context"
	"filmmash/internal/config"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"filmmash/internal/vote"
	"filmmash/internal/web"
	"fmt"
	"log"
	"os"
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

	m := metrics.NewMetrics()

	filmService := film.NewService(pool)
	duelService := duel.NewService(pool, filmService)

	voteRepo := vote.NewRepository(pool)
	registerVoteUC := vote.NewRegisterVoteUC(voteRepo, filmService, duelService)

	router := web.InitRouter(m, filmService, duelService, registerVoteUC)

	_, err = InitServer(cfg.Port, router)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
