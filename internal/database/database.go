package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const rollbackTimeout = 5 * time.Second

type DBConn interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type txKey struct{}

func ExtractTx(ctx context.Context, pool *pgxpool.Pool) DBConn {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	if ok {
		return tx
	}
	return pool
}

func InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func rollback(ctx context.Context, tx pgx.Tx, method string) {
	rbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()

	if rbErr := tx.Rollback(rbCtx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
		slog.ErrorContext(rbCtx, "failed to rollback transaction",
			slog.String("method", method),
			slog.String("error", rbErr.Error()),
		)
	}
}

func (tm *TxManager) ExecTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			rollback(ctx, tx, "ExecTx")
			panic(p)
		} else if err != nil {
			rollback(ctx, tx, "ExecTx")
		} else {
			err = tx.Commit(ctx)
		}
	}()
	txCtx := context.WithValue(ctx, txKey{}, tx)
	err = fn(txCtx)
	return err
}

func (tm *TxManager) Begin(ctx context.Context) (pgx.Tx, error) {
	return tm.pool.Begin(ctx)
}

func NewTx(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	return pool.Begin(ctx)
}

func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(ctx, tx, "WithTx")

	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	cfg, err := pgxpool.ParseConfig(dataSourceName)
	if err != nil {
		return nil, err
	}
	cfg.MinConns = 1
	return pgxpool.NewWithConfig(ctx, cfg)
}
