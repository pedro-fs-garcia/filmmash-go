-- +goose Up
ALTER TABLE films ADD COLUMN imdb_id TEXT CHECK (imdb_id ~ '^tt[0-9]+$');
ALTER TABLE reel_alternatives DROP CONSTRAINT reel_alternatives_seq_check;
ALTER TABLE reel_alternatives ADD CHECK (seq BETWEEN 1 AND 5);
ALTER TABLE frames ADD COLUMN curated_order SMALLINT CHECK (curated_order BETWEEN 1 AND 4);
ALTER TABLE frames ADD COLUMN approved BOOLEAN NOT NULL DEFAULT FALSE;
CREATE UNIQUE INDEX frames_curated_order ON frames(film_id, curated_order) WHERE approved;
CREATE TABLE freeze_frame_plays (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    game_id INTEGER NOT NULL REFERENCES games(id),
    player_key TEXT NOT NULL,
    user_id UUID REFERENCES users(id),
    progress JSONB NOT NULL DEFAULT '{}',
    score INTEGER NOT NULL DEFAULT 0 CHECK (score BETWEEN 0 AND 1000),
    finished BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    UNIQUE (game_id, player_key)
);
CREATE UNIQUE INDEX freeze_frame_account_day ON freeze_frame_plays(game_id,user_id) WHERE user_id IS NOT NULL;
CREATE INDEX freeze_frame_history ON freeze_frame_plays(user_id,game_id) WHERE finished;

-- +goose Down
DROP TABLE freeze_frame_plays;
DROP INDEX frames_curated_order;
ALTER TABLE frames DROP COLUMN approved, DROP COLUMN curated_order;
-- Keep the expanded option constraint so rollback does not reject existing games.
ALTER TABLE films DROP COLUMN imdb_id;
