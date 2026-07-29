package main

import (
	"context"
	"filmmash/internal/config"
	"filmmash/internal/database"
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
		"postgres://%s:%s@%s:%s/%s?sslmode=require",
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
	slog.SetDefault(logger)

	pool, err := database.Pool(ctx, dsn)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create database pool of connection", slog.String("error", err.Error()))
		panic(err)
	}

	router, err := initRouter(ctx, cfg, pool, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to wire router", slog.String("error", err.Error()))
		panic(err)
	}

	httpServer, err := InitServer(cfg.Port, router)
	if err != nil {
		logger.ErrorContext(ctx, "failed to start server", slog.String("error", err.Error()))
		panic(err)
	}

	logger.InfoContext(ctx, fmt.Sprintf("server listening on port %s\n", httpServer.Addr))
	return nil
}
