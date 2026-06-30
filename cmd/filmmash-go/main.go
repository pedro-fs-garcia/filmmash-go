package main

import (
	"context"
	"filmmash/internal/auth"
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

	filmService := film.NewService(pool)

	duelService := duel.NewService(m.DuelMetrics, pool, filmService)

	voteRepo := vote.NewRepository(pool)
	registerVoteUC := vote.NewRegisterVoteUC(m.VoteMetrics, voteRepo, filmService, duelService)
	listVoteUC := vote.NewListVotesUC(voteRepo)

	provider := auth.NewZitadelProvider(
		cfg.ZitadelClientId,
		cfg.ZitadelClientSecret,
		cfg.ZitadelBaseURL,
	)

	sessionRepo := auth.NewRepository(pool)
	authService := auth.NewService(provider, sessionRepo)

	authHandler := auth.NewHandler(
		logger.With("component", "AuthHandler"),
		provider,
		authService,
	)
	filmHandler := film.NewHandler(logger.With("package", "film"), filmService)
	duelHandler := duel.NewHandler(logger.With("package", "duel"), duelService)
	voteHandler := vote.NewHandler(logger.With("package", "vote"), registerVoteUC, listVoteUC)

	router := web.InitRouter(logger.With("package", "web"), m, authService, authHandler, filmHandler, duelHandler, voteHandler)

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

	totalVotes, err := voteRepo.CurrentTotal(ctx)
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
