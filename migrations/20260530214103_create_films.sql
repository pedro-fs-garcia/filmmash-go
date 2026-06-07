-- +goose Up
SELECT 'up SQL query';

CREATE TABLE directors (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(name) <= 127)
);

CREATE TABLE films (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL CHECK (length(title) <= 255),
    release_year SMALLINT NOT NULL CHECK (release_year BETWEEN 1800 AND 2100),
    director_id INTEGER NOT NULL REFERENCES directors(id) ON DELETE RESTRICT,
    image_path TEXT,
    rating FLOAT NOT NULL DEFAULT 1400,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_films_director_id ON films(director_id);

CREATE TABLE duels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    film_a_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    film_b_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (film_a_id <> film_b_id)
);

CREATE SEQUENCE vote_seq;

CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    duel_id UUID NOT NULL UNIQUE REFERENCES duels(id) ON DELETE RESTRICT,
    winner_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    loser_id INT NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    winner_rating_after FLOAT NOT NULL,
    loser_rating_after FLOAT NOT NULL,
    applied_seq BIGINT NOT NULL DEFAULT NEXTVAL('vote_seq'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (winner_id <> loser_id)
);


-- +goose Down
SELECT 'down SQL query';

DROP TABLE votes;
DROP SEQUENCE vote_seq;
DROP TABLE duels;
DROP INDEX idx_films_director_id;
DROP TABLE films;
DROP TABLE directors;
