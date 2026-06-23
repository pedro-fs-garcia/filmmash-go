package auth

import (
	"context"
	"filmmash/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionEventRepository struct {
	pool *pgxpool.Pool
}

func NewSessionEventRepository(pool *pgxpool.Pool) *SessionEventRepository {
	return &SessionEventRepository{
		pool: pool,
	}
}

func (r *SessionEventRepository) Insert(ctx context.Context, se *SessionEvent) error {
	query := `
	INSERT INTO session_events (session_id, user_id, event, ip_address)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(
		ctx, query, se.SessionID, se.UserID, se.Event, se.IPAddress,
	).Scan(&se.ID, &se.CreatedAt)

	if err != nil {
		return database.ParseDBError("inserting to session_events", err)
	}
	return nil
}
