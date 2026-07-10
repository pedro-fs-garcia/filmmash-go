package admin

import (
	"context"
	"filmmash/internal/database"

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
	query := `
		SELECT u.id, u.idp_sub, u.created_at, COUNT(*) votes
		FROM users u
		JOIN votes v ON u.id = v.user_id
		WHERE u.id > $1
		GROUP BY u.id, u.idp_sub, u.created_at
		ORDER BY u.id
		LIMIT $2
	`
	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, pars.LastSeenId, pars.Size)
	if err != nil {
		return nil, database.ParseDBError("querying paginated users", err)
	}
	defer rows.Close()

	var users []UserWithVote
	for rows.Next() {
		var u UserWithVote
		err := rows.Scan(&u.Id, &u.PidSub, &u.CreatedAt, &u.Votes)
		if err != nil {
			return nil, database.ParseDBError("scanning film row", err)
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		return nil, database.ParseDBError("iterating rows", err)
	}
	return users, nil
}
