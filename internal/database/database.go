package database

import (
	"context"
	"database/sql"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBConn interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

func OpenDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
		panic(err)
	}

	return db, nil
}

func Pool(ctx context.Context, dataSourceName string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dataSourceName)
	return pool, err
}
