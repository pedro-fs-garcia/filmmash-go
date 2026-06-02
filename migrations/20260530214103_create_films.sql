-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS directors (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(name) <= 127)
);

CREATE TABLE IF NOT EXISTS films (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL CHECK (length(title) <= 255),
    release_year SMALLINT NOT NULL CHECK (release_year BETWEEN 1800 AND 2100),
    director_id INTEGER NOT NULL REFERENCES directors(id) ON DELETE RESTRICT,
    image_path TEXT,
    rating INTEGER NOT NULL DEFAULT 1400,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_films_director_id ON films(director_id);

-- +goose Down
SELECT 'down SQL query';

DROP INDEX idx_films_director_id;
DROP TABLE films;
DROP TABLE directors;
