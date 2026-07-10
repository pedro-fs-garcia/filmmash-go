package vote

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

func (r *repository) queries(ctx context.Context) *dbgen.Queries {
	return dbgen.New(database.ExtractTx(ctx, r.pool))
}

func (r *repository) InsertVote(ctx context.Context, vote *Vote) error {
	id, err := r.queries(ctx).InsertVote(ctx, dbgen.InsertVoteParams{
		DuelID:            vote.DuelId,
		UserID:            vote.UserID,
		WinnerID:          int32(vote.WinnerID),
		LoserID:           int32(vote.LoserId),
		WinnerRatingAfter: vote.WinnerRatingAfter,
		LoserRatingAfter:  vote.LoserRatingAfter,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("[vote.repository.InsertVote] conflict inserting vote: %w", ErrDuplicateEntry)
		}
		return fmt.Errorf("[vote.repository.InsertVote] Failed to insert vote: %w", err)
	}
	vote.Id = id
	return nil
}

func (r *repository) ListFilmVotes(ctx context.Context, filmId int) ([]MatchupResult, error) {
	rows, err := r.queries(ctx).ListFilmVotes(ctx, int32(filmId))
	if err != nil {
		return nil, database.ParseDBError("querying list of film votes", err)
	}

	var votes []MatchupResult
	for _, row := range rows {
		votes = append(votes, MatchupResult{
			Winner: VotedFilm{
				Id:          int(row.WinnerID),
				Title:       row.WinnerTitle,
				Year:        int(row.WinnerReleaseYear),
				RatingAfter: row.WinnerRatingAfter,
			},
			Loser: VotedFilm{
				Id:          int(row.LoserID),
				Title:       row.LoserTitle,
				Year:        int(row.LoserReleaseYear),
				RatingAfter: row.LoserRatingAfter,
			},
			CreatedAt: row.CreatedAt,
		})
	}
	return votes, nil
}

func (r *repository) CurrentTotal(ctx context.Context) (int, error) {
	n, err := r.queries(ctx).CountVotes(ctx)
	if err != nil {
		return 0, fmt.Errorf("[vote.repository.CurrentTotal] failed to count total current votes: %w", err)
	}
	return int(n), nil
}

func (r *repository) ListUsersVotes(ctx context.Context, userId uuid.UUID) ([]MatchupResult, error) {
	rows, err := r.queries(ctx).ListUserVotes(ctx, &userId)
	if err != nil {
		return nil, database.ParseDBError("querying list of film votes", err)
	}

	var votes []MatchupResult
	for _, row := range rows {
		winner := VotedFilm{
			Id:          int(row.WinnerID),
			Title:       row.WinnerTitle,
			Year:        int(row.WinnerReleaseYear),
			RatingAfter: row.WinnerRatingAfter,
		}
		if row.WinnerImagePath != nil {
			winner.ImagePath = *row.WinnerImagePath
		}
		loser := VotedFilm{
			Id:          int(row.LoserID),
			Title:       row.LoserTitle,
			Year:        int(row.LoserReleaseYear),
			RatingAfter: row.LoserRatingAfter,
		}
		if row.LoserImagePath != nil {
			loser.ImagePath = *row.LoserImagePath
		}
		votes = append(votes, MatchupResult{
			Winner:    winner,
			Loser:     loser,
			CreatedAt: row.CreatedAt,
		})
	}
	return votes, nil
}
