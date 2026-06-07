package duel

import (
	"context"
	"filmmash/internal/film"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func newRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

func (r *repository) Insert(ctx context.Context, filmA, filmB *film.Film) (uuid.UUID, error) {
	query := `INSERT INTO duels (film_a_id, film_b_id) VALUES ($1, $2) RETURNING id`
	var duelId uuid.UUID
	err := r.pool.QueryRow(ctx, query, filmA.Id, filmB.Id).Scan(&duelId)
	if err != nil {
		log.Println(err)
		return uuid.Nil, err
	}
	return duelId, nil
}
