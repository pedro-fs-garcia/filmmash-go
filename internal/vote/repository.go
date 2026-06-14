package vote

import (
	"context"
	"filmmash/internal/database"

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
	q := database.ExtractTx(ctx, r.pool)
	return q.QueryRow(ctx, query,
		vote.DuelId,
		vote.WinnerID, vote.LoserId, vote.WinnerRatingAfter, vote.LoserRatingAfter,
	).Scan(&vote.Id)
}

func (r *repository) CurrrentTotal(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM votes"
	var n int
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query).Scan(&n)
	return n, err
}
