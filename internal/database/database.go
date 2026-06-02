package database

import (
	"context"
	"database/sql"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenDB(dataSourceName string) *sql.DB {
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
		panic(err)
	}

	return db
}

func Pool(ctx context.Context, dataSourceName string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dataSourceName)
	return pool, err
}
