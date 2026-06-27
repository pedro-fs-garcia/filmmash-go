package vote

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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
	INSERT INTO votes (duel_id, user_id, winner_id, loser_id, winner_rating_after, loser_rating_after) 
	VALUES ($1, $2, $3, $4, $5, $6) 
	RETURNING id
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query,
		vote.DuelId, vote.UserID,
		vote.WinnerID, vote.LoserId, vote.WinnerRatingAfter, vote.LoserRatingAfter,
	).Scan(&vote.Id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("[vote.repository.InsertVote] conflict inserting vote: %w", ErrDuplicateEntry)
		}
		return fmt.Errorf("[vote.repository.InsertVote] Failed to insert vote: %w", err)
	}
	return nil
}

func (r *repository) CurrentTotal(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM votes"
	var n int
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("[vote.repository.CurrentTotal] failed to count total current votes: %w", err)
	}
	return n, nil
}
