-- name: UpsertUser :one
INSERT INTO users (idp_sub) VALUES ($1)
ON CONFLICT (idp_sub) DO UPDATE SET idp_sub = EXCLUDED.idp_sub
RETURNING id, created_at;

-- name: GetUserBySub :one
SELECT id, idp_sub, created_at FROM users WHERE idp_sub = $1;

-- name: GetUserByID :one
SELECT id, idp_sub, created_at FROM users WHERE id = $1;

-- name: InsertSession :one
INSERT INTO sessions (
    token_hash, user_id, access_token, access_token_expires_at, refresh_token,
    id_token, scopes, ip_address, user_agent, roles, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id;

-- name: GetSessionByTokenHash :one
SELECT * FROM sessions WHERE token_hash = $1 AND expires_at > now();

-- name: DeleteSessionByTokenHash :execrows
DELETE FROM sessions WHERE token_hash = $1;

-- name: InsertSessionEvent :one
INSERT INTO session_events (session_id, idp_sub, event, ip_address)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at;

-- name: DeleteExpiredSessions :execrows
WITH expired_sessions AS (
    SELECT id FROM sessions
    WHERE expires_at < CURRENT_TIMESTAMP
    LIMIT 5000
    FOR UPDATE SKIP LOCKED
)
DELETE FROM sessions WHERE id IN (SELECT id FROM expired_sessions);
