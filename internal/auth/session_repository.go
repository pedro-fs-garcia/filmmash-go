package auth

import (
	"context"
	"filmmash/internal/database"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		pool: pool,
	}
}

func (r *SessionRepository) UpsertUser(ctx context.Context, user *User) error {
	query := `
	INSERT INTO users (zitadel_sub) VALUES ($1)
	ON CONFLICT (zitadel_sub) DO UPDATE SET zitadel_sub = EXCLUDED.zitadel_sub
	RETURNING id, created_at;
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query, user.ZitadelSub).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return database.ParseDBError("upserting user", err)
	}
	return nil
}

func (r *SessionRepository) GetUserBySub(ctx context.Context, sub string) (User, error) {
	query := "SELECT id, zitadel_sub, created_at FROM users WHERE zitadel_sub = $1"
	q := database.ExtractTx(ctx, r.pool)
	var user User
	err := q.QueryRow(ctx, query, sub).Scan(&user.ID, &user.ZitadelSub, &user.CreatedAt)
	if err != nil {
		return User{}, database.ParseDBError("getting user by sub", err)
	}
	return user, nil
}

func (r *SessionRepository) GetUserById(ctx context.Context, id uuid.UUID) (User, error) {
	query := "SELECT id, zitadel_sub, created_at FROM users WHERE id = $1"
	q := database.ExtractTx(ctx, r.pool)
	var user User
	err := q.QueryRow(ctx, query, id).Scan(&user.ID, &user.ZitadelSub, &user.CreatedAt)
	if err != nil {
		return User{}, database.ParseDBError("getting user by id", err)
	}
	return user, nil
}

func (r *SessionRepository) Insert(ctx context.Context, session *SessionDB) error {
	const query = `
	INSERT INTO sessions (
		token_hash, user_id, access_token, access_token_expires_at, refresh_token, 
		id_token, scopes, ip_address, user_agent, expires_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		session.ExpiresAt,
	).Scan(&session.ID)

	return database.ParseDBError("inserting session", err)
}

func (r *SessionRepository) GetByTokenHash(ctx context.Context, tokenHash []byte) (Session, error) {
	query := `
	SELECT id, token_hash, user_id, access_token, access_token_expires_at,
	       refresh_token, id_token, scopes, ip_address, user_agent,
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

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash []byte) error {
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

func SessionToSessionDB(session Session, tokenHash []byte) SessionDB {
	scope := strings.Join(session.Scopes, " ")
	sessionDB := SessionDB{
		ID:                   session.ID,
		TokenHash:            tokenHash,
		UserID:               session.UserID,
		AccessToken:          []byte(session.AccessToken),
		AccessTokenExpiresAt: &session.AccessTokenExpiresAt,
		RefreshToken:         []byte(session.RefreshToken),
		IDToken:              []byte(session.IDToken),
		Scopes:               &scope,
		IPAddress:            session.IPAddress,
		UserAgent:            &session.UserAgent,
		CreatedAt:            session.CreatedAt,
		LastSeenAt:           session.LastSeenAt,
		ExpiresAt:            session.ExpiresAt,
	}
	if len(scope) == 0 {
		sessionDB.Scopes = nil
	}
	return sessionDB
}

func SessionDBToSession(sdb SessionDB) Session {
	session := Session{
		ID:           sdb.ID,
		UserID:       sdb.UserID,
		AccessToken:  string(sdb.AccessToken),
		RefreshToken: string(sdb.RefreshToken),
		IDToken:      string(sdb.IDToken),
		IPAddress:    sdb.IPAddress,
		CreatedAt:    sdb.CreatedAt,
		LastSeenAt:   sdb.LastSeenAt,
		ExpiresAt:    sdb.ExpiresAt,
	}
	if sdb.AccessTokenExpiresAt != nil {
		session.AccessTokenExpiresAt = *sdb.AccessTokenExpiresAt
	}
	if sdb.Scopes != nil {
		session.Scopes = strings.Split(*sdb.Scopes, " ")
	}
	if sdb.UserAgent != nil {
		session.UserAgent = *sdb.UserAgent
	}
	return session
}
