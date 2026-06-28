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

func (r *repository) ListFilmVotes(ctx context.Context, filmId int) ([]MatchupResult, error) {
	query := `
	SELECT v.winner_id, v.loser_id, v.winner_rating_after, v.loser_rating_after, v.created_at,
		fw.title, fw.release_year, fl.title, fl.release_year
	FROM votes v
	JOIN films fw ON fw.id = v.winner_id
	JOIN films fl ON fl.id = v.loser_id
	WHERE v.winner_id = $1 OR v.loser_id = $1
	ORDER BY v.created_at DESC
	`
	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, filmId)
	if err != nil {
		return nil, database.ParseDBError("querying list of film votes", err)
	}

	var votes []MatchupResult
	for rows.Next() {
		var vr MatchupResult
		err = rows.Scan(
			&vr.Winner.Id, &vr.Loser.Id, &vr.Winner.RatingAfter, &vr.Loser.RatingAfter,
			&vr.CreatedAt,
			&vr.Winner.Title, &vr.Winner.Year, &vr.Loser.Title, &vr.Loser.Year,
		)
		votes = append(votes, vr)
	}
	if err = rows.Err(); err != nil {
		return nil, database.ParseDBError("iterating rows", err)
	}
	return votes, nil
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
