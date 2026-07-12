-- +goose Up
CREATE TABLE directors (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(name) BETWEEN 2 AND 127)
);

CREATE TABLE films (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL CHECK (length(title) <= 255),
    release_year SMALLINT NOT NULL CHECK (release_year BETWEEN 1800 AND 2100),
    director_id INTEGER NOT NULL REFERENCES directors(id) ON DELETE RESTRICT,
    image_path TEXT,
    popularity FLOAT NOT NULL DEFAULT 0 CHECK (popularity >= 0),
    vote_average FLOAT NOT NULL DEFAULT 0 CHECK (vote_average >= 0),
    rating FLOAT NOT NULL DEFAULT 1400,
    duel_count INT NOT NULL DEFAULT 0 CHECK (duel_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_films_director_id ON films(director_id);
CREATE INDEX idx_films_rating_id ON films(rating, id);
CREATE INDEX idx_films_rating ON films(rating);
CREATE INDEX idx_films_popularity ON films(popularity);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON films
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

CREATE TABLE duels (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    film_a_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    film_b_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (film_a_id <> film_b_id)
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    idp_sub TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE SEQUENCE vote_seq;

CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    duel_id UUID NOT NULL UNIQUE REFERENCES duels(id) ON DELETE RESTRICT,
    winner_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    loser_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    winner_rating_after FLOAT NOT NULL,
    loser_rating_after FLOAT NOT NULL,
    applied_seq BIGINT NOT NULL DEFAULT NEXTVAL('vote_seq'),
    user_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (winner_id <> loser_id)
);

CREATE INDEX idx_votes_user_id ON votes(user_id);

CREATE TABLE sessions (
    id                        UUID PRIMARY KEY DEFAULT uuidv7(),        -- internal PK. NOT the cookie value.
    token_hash                BYTEA NOT NULL UNIQUE,   -- SHA-256 of the opaque session token. Raw token lives only in the cookie.
    user_id                   UUID NOT NULL CONSTRAINT fk_sessions_user REFERENCES users(id),
    access_token              BYTEA,                   -- encrypted at rest. Only if you call Zitadel/downstream APIs as the user.
    access_token_expires_at   timestamptz,             -- drives refresh.
    refresh_token             BYTEA,                   -- encrypted. Requires `offline_access` scope. Most sensitive field here.
    id_token                  BYTEA,                   -- encrypted. Kept for `id_token_hint` at RP-initiated logout.
    scopes                    TEXT,                    -- granted scopes (audit / capability check).
    ip_address                inet,                    -- audit / anomaly detection. Do NOT hard-bind sessions to it.
    user_agent                TEXT,                    -- audit.
    roles                     TEXT[] NOT NULL DEFAULT '{}',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at              TIMESTAMPTZ NOT NULL DEFAULT now(),  -- idle timeout.
    expires_at                TIMESTAMPTZ NOT NULL     -- absolute lifetime cap.
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE session_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_id UUID NOT NULL,
    idp_sub TEXT NOT NULL,
    event TEXT NOT NULL,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_events_idp_sub ON session_events(idp_sub);
CREATE INDEX idx_session_events_session_id_idp_sub ON session_events(session_id, idp_sub);

-- +goose Down
DROP TABLE session_events;
DROP TABLE sessions;
DROP TABLE votes;
DROP SEQUENCE vote_seq;
DROP TABLE users;
DROP TABLE duels;
DROP TRIGGER IF EXISTS set_timestamp ON films;
DROP FUNCTION trigger_set_timestamp();
DROP TABLE films;
DROP TABLE directors;
