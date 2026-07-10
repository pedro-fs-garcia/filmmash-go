package auth

import (
	"context"
	"filmmash/internal/database"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *repository) UpsertUser(ctx context.Context, user *User) error {
	query := `
	INSERT INTO users (idp_sub) VALUES ($1)
	ON CONFLICT (idp_sub) DO UPDATE SET idp_sub = EXCLUDED.idp_sub
	RETURNING id, created_at;
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query, user.PidSub).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return database.ParseDBError("upserting user", err)
	}
	return nil
}

func (r *repository) GetUserBySub(ctx context.Context, sub string) (User, error) {
	query := "SELECT id, idp_sub, created_at FROM users WHERE idp_sub = $1"
	q := database.ExtractTx(ctx, r.pool)
	var user User
	err := q.QueryRow(ctx, query, sub).Scan(&user.ID, &user.PidSub, &user.CreatedAt)
	if err != nil {
		return User{}, database.ParseDBError("getting user by sub", err)
	}
	return user, nil
}

func (r *repository) GetUserById(ctx context.Context, id uuid.UUID) (User, error) {
	query := "SELECT id, idp_sub, created_at FROM users WHERE id = $1"
	q := database.ExtractTx(ctx, r.pool)
	var user User
	err := q.QueryRow(ctx, query, id).Scan(&user.ID, &user.PidSub, &user.CreatedAt)
	if err != nil {
		return User{}, database.ParseDBError("getting user by id", err)
	}
	return user, nil
}

func (r *repository) Insert(ctx context.Context, session *SessionDB) error {
	const query = `
	INSERT INTO sessions (
		token_hash, user_id, access_token, access_token_expires_at, refresh_token, 
		id_token, scopes, ip_address, user_agent, roles, expires_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING id
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query,
		session.TokenHash,
		session.UserID,
		session.AccessToken,
		session.AccessTokenExpiresAt,
		session.RefreshToken,
		session.IDToken,
		session.Scopes,
		session.IPAddress,
		session.UserAgent,
		session.Roles,
		session.ExpiresAt,
	).Scan(&session.ID)

	return database.ParseDBError("inserting session", err)
}

func (r *repository) GetByTokenHash(ctx context.Context, tokenHash []byte) (Session, error) {
	query := `
	SELECT id, token_hash, user_id, access_token, access_token_expires_at,
	       refresh_token, id_token, scopes, ip_address, user_agent, roles,
	       created_at, last_seen_at, expires_at
	FROM sessions WHERE token_hash = $1`

	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, tokenHash)
	if err != nil {
		return Session{}, database.ParseDBError("querying session by token_hash", err)
	}
	sessionDB, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[SessionDB])
	if err != nil {
		return Session{}, database.ParseDBError("querying session by token_hash", err)
	}
	return SessionDBToSession(sessionDB), nil
}

func (r *repository) DeleteByTokenHash(ctx context.Context, tokenHash []byte) error {
	query := "DELETE FROM sessions WHERE token_hash = $1"
	q := database.ExtractTx(ctx, r.pool)
	tag, err := q.Exec(ctx, query, tokenHash)
	if err != nil {
		return database.ParseDBError("deleting sesison by token_hash", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("No rows affected %w", ErrSessionNotFound)
	}
	return nil
}

func (r *repository) InsertEvent(ctx context.Context, se *SessionEvent) error {
	query := `
	INSERT INTO session_events (session_id, idp_sub, event, ip_address)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(
		ctx, query, se.SessionID, se.PidSub, se.Event, se.IPAddress,
	).Scan(&se.ID, &se.CreatedAt)

	if err != nil {
		return database.ParseDBError("inserting to session_events", err)
	}
	return nil
}

func (r *repository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	query := `
		WITH expired_sessions AS (
			SELECT id FROM sessions 
			WHERE expires_at < CURRENT_TIMESTAMP 
			LIMIT 5000 
			FOR UPDATE SKIP LOCKED
		)
		DELETE FROM sessions WHERE id IN (SELECT id FROM expired_sessions)
	`
	var total int64

	// Do not allow context-injected transactions to wrap this operation
	pool := r.pool
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		tag, err := pool.Exec(ctx, query)
		if err != nil {
			return 0, database.ParseDBError("deleting expired sessions", err)
		}
		if tag.RowsAffected() == 0 {
			break
		}
		total += tag.RowsAffected()

		// Throttle execution to prevent IO/CPU spikes
		time.Sleep(50 * time.Millisecond)
	}
	return total, nil
}
