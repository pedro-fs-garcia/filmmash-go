package testdb

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

type config struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
}

func projectRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot resolve caller path")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found walking up from " + file)
		}
		dir = parent
	}
}

func New(ctx context.Context) (*pgxpool.Pool, func(), error) {
	root, err := projectRoot()
	if err != nil {
		return nil, nil, err
	}

	envPath := filepath.Join(root, ".env.test")
	if _, statErr := os.Stat(envPath); statErr == nil {
		if err := godotenv.Overload(envPath); err != nil {
			return nil, nil, fmt.Errorf("Error loading .env.test: %w", err)
		}
	} else if os.Getenv("DATABASE_URL") == "" {
		return nil, nil, fmt.Errorf(".env.test not found at %s and DATABASE_URL not set", envPath)
	}

	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return nil, nil, err
	}

	if !strings.Contains(cfg.DatabaseURL, "test") {
		return nil, nil, fmt.Errorf("refusing to run: DATABASE_URL %q is not a test database", cfg.DatabaseURL)
	}

	db, err := database.OpenDB(cfg.DatabaseURL)
	goose.SetDialect("postgres")
	if err := goose.Up(db, filepath.Join(root, "migrations")); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("Error applying migrations: %w", err)
	}
	db.Close()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, err
	}

	return pool, func() { pool.Close() }, nil
}

func Truncate(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	_, err := pool.Exec(ctx,
		fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", ")),
	)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
