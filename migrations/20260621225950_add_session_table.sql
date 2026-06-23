-- +goose Up
SELECT 'up SQL query';

CREATE TABLE sessions (
    id                        UUID PRIMARY KEY DEFAULT uuidv7(),        -- internal PK. NOT the cookie value.
    token_hash                BYTEA NOT NULL UNIQUE,   -- SHA-256 of the opaque session token. Raw token lives only in the cookie.
    user_id                   TEXT NOT NULL,           -- Zitadel `sub`. FK to your users table if you provision locally.
    access_token              BYTEA,                   -- encrypted at rest. Only if you call Zitadel/downstream APIs as the user.
    access_token_expires_at   timestamptz,             -- drives refresh.
    refresh_token             BYTEA,                   -- encrypted. Requires `offline_access` scope. Most sensitive field here.
    id_token                  BYTEA,                   -- encrypted. Kept for `id_token_hint` at RP-initiated logout.
    scopes                    TEXT,                    -- granted scopes (audit / capability check).
    ip_address                inet,                    -- audit / anomaly detection. Do NOT hard-bind sessions to it.
    user_agent                TEXT,                    -- audit.
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at              TIMESTAMPTZ NOT NULL DEFAULT now(),  -- idle timeout.
    expires_at                TIMESTAMPTZ NOT NULL     -- absolute lifetime cap.
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);


-- +goose Down
SELECT 'down SQL query';

DROP INDEX idx_sessions_user_id;
DROP INDEX idx_sessions_expires_at;
DROP TABLE sessions;
