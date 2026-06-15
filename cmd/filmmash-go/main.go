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
	"log/slog"
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

	logger := slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		},
	))

	pool, err := database.Pool(ctx, dsn)
	if err != nil {
		panic(err)
	}

	m := metrics.NewMetrics()

	filmLogger := logger.With(slog.String("package", "film"))
	filmService := film.NewService(filmLogger, pool)

	// duelLogger := logger.With(slog.String("package", "duel"))
	duelService := duel.NewService(pool, filmService)

	voteRepo := vote.NewRepository(pool)
	registerVoteUC := vote.NewRegisterVoteUC(m.VoteMetrics, voteRepo, filmService, duelService)

	router := web.InitRouter(m, filmService, duelService, registerVoteUC)

	pendingDuels, err := duelService.CountPending(context.Background())
	if err != nil {
		panic(err)
	}
	m.DuelMetrics.SeedPending(pendingDuels)

	totalFilms, err := filmService.CountTotal(ctx)
	if err != nil {
		panic(err)
	}
	m.FilmMetrics.SeedCurrentTotal(totalFilms)

	totalVotes, err := voteRepo.CurrrentTotal(ctx)
	if err != nil {
		panic(err)
	}
	m.VoteMetrics.SeedTotal(totalVotes)

	_, err = InitServer(cfg.Port, router)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
