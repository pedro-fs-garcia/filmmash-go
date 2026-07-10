package admin

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

func (r *repository) GetUsersPaginated(ctx context.Context, pars PaginationParameters) ([]UserWithVote, error) {
	lastSeenId, err := uuid.Parse(pars.LastSeenId)
	if err != nil {
		return nil, fmt.Errorf("parsing last_seen_id (%q): %w", pars.LastSeenId, database.ErrInvalidInput)
	}

	q := dbgen.New(database.ExtractTx(ctx, r.pool))
	rows, err := q.ListUsersWithVotes(ctx, dbgen.ListUsersWithVotesParams{
		ID:    lastSeenId,
		Limit: int32(pars.Size),
	})
	if err != nil {
		return nil, database.ParseDBError("querying paginated users", err)
	}

	var users []UserWithVote
	for _, row := range rows {
		users = append(users, UserWithVote{
			Id:        row.ID,
			PidSub:    row.PidSub,
			CreatedAt: row.CreatedAt,
			Votes:     row.Votes,
		})
	}
	return users, nil
}
