package duel

import (
	"context"
	"filmmash/internal/film"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Duel struct {
	Id    uuid.UUID
	FilmA *film.Film
	FilmB *film.Film
}

type FilmRating struct {
	Id     int
	Rating float64
}

type DBConn interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}
