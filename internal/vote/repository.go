package vote

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

func (r *repository) InsertVote(ctx context.Context, vote *Vote) error {
	query := `
	INSERT INTO votes (duel_id, winner_id, loser_id, winner_rating_after, loser_rating_after) 
	VALUES ($1, $2, $3, $4, $5) 
	RETURNING id
	`
	return r.pool.QueryRow(ctx, query,
		vote.DuelId,
		vote.WinnerID, vote.LoserId, vote.WinnerRatingAfter, vote.LoserRatingAfter,
	).Scan(&vote.Id)
}
