package auth

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"fmt"
	"time"

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

func (r *repository) queries(ctx context.Context) *dbgen.Queries {
	return dbgen.New(database.ExtractTx(ctx, r.pool))
}

func (r *repository) UpsertUser(ctx context.Context, user *User) error {
	row, err := r.queries(ctx).UpsertUser(ctx, user.PidSub)
	if err != nil {
		return database.ParseDBError("upserting user", err)
	}
	user.ID = row.ID
	user.CreatedAt = row.CreatedAt
	return nil
}

func (r *repository) GetUserBySub(ctx context.Context, sub string) (User, error) {
	row, err := r.queries(ctx).GetUserBySub(ctx, sub)
	if err != nil {
		return User{}, database.ParseDBError("getting user by sub", err)
	}
	return User(row), nil
}

func (r *repository) GetUserById(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.queries(ctx).GetUserByID(ctx, id)
	if err != nil {
		return User{}, database.ParseDBError("getting user by id", err)
	}
	return User(row), nil
}

func (r *repository) Insert(ctx context.Context, session *SessionDB) error {
	id, err := r.queries(ctx).InsertSession(ctx, dbgen.InsertSessionParams{
		TokenHash:            session.TokenHash,
		UserID:               session.UserID,
		AccessToken:          session.AccessToken,
		AccessTokenExpiresAt: session.AccessTokenExpiresAt,
		RefreshToken:         session.RefreshToken,
		IDToken:              session.IDToken,
		Scopes:               session.Scopes,
		IPAddress:            session.IPAddress,
		UserAgent:            session.UserAgent,
		Roles:                session.Roles,
		ExpiresAt:            session.ExpiresAt,
	})
	if err != nil {
		return database.ParseDBError("inserting session", err)
	}
	session.ID = id
	return nil
}

func (r *repository) GetByTokenHash(ctx context.Context, tokenHash []byte) (Session, error) {
	row, err := r.queries(ctx).GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return Session{}, database.ParseDBError("querying session by token_hash", err)
	}
	return SessionDBToSession(SessionDB(row)), nil
}

func (r *repository) DeleteByTokenHash(ctx context.Context, tokenHash []byte) error {
	affected, err := r.queries(ctx).DeleteSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return database.ParseDBError("deleting sesison by token_hash", err)
	}
	if affected == 0 {
		return fmt.Errorf("no rows affected %w", ErrSessionNotFound)
	}
	return nil
}

func (r *repository) InsertEvent(ctx context.Context, se *SessionEvent) error {
	row, err := r.queries(ctx).InsertSessionEvent(ctx, dbgen.InsertSessionEventParams{
		SessionID: se.SessionID,
		PidSub:    se.PidSub,
		Event:     string(se.Event),
		IPAddress: se.IPAddress,
	})
	if err != nil {
		return database.ParseDBError("inserting to session_events", err)
	}
	se.ID = row.ID
	se.CreatedAt = row.CreatedAt
	return nil
}

func (r *repository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	var total int64

	// Do not allow context-injected transactions to wrap this operation
	q := dbgen.New(r.pool)
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		affected, err := q.DeleteExpiredSessions(ctx)
		if err != nil {
			return 0, database.ParseDBError("deleting expired sessions", err)
		}
		if affected == 0 {
			break
		}
		total += affected

		// Throttle execution to prevent IO/CPU spikes
		time.Sleep(50 * time.Millisecond)
	}
	return total, nil
}
