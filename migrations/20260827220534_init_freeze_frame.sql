-- +goose Up

CREATE TABLE games (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    valid_at   DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE reels (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    game_id    INTEGER NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    film_id    INTEGER NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    seq        SMALLINT NOT NULL CHECK (seq BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (game_id, seq),
    UNIQUE (game_id, film_id)
);

CREATE INDEX idx_reels_film_id ON reels(film_id);

CREATE TABLE reel_alternatives (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reel_id    INTEGER NOT NULL REFERENCES reels(id) ON DELETE RESTRICT,
    film_id    INTEGER NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    seq        SMALLINT NOT NULL CHECK (seq BETWEEN 1 AND 4),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, reel_id),   -- required for the composite FK in answers
    UNIQUE (reel_id, seq),
    UNIQUE (reel_id, film_id)
);

CREATE INDEX idx_reel_alternatives_film_id ON reel_alternatives(film_id);

CREATE TABLE frames (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    film_id    INTEGER NOT NULL REFERENCES films(id) ON DELETE RESTRICT,
    image_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_frames_film_id ON frames(film_id);

CREATE TABLE reel_frames (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reel_id    INTEGER NOT NULL REFERENCES reels(id) ON DELETE RESTRICT,
    frame_id   INTEGER NOT NULL REFERENCES frames(id) ON DELETE RESTRICT,
    difficulty SMALLINT NOT NULL CHECK (difficulty BETWEEN 1 AND 10), -- feeds the multiplier, not the reveal order
    seq        SMALLINT NOT NULL CHECK (seq BETWEEN 1 AND 5),
    UNIQUE (reel_id, frame_id),
    UNIQUE (reel_id, seq)
);

CREATE INDEX idx_reel_frames_frame_id ON reel_frames(frame_id);

CREATE TABLE answers (
    id                  INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reel_id             INTEGER NOT NULL REFERENCES reels(id) ON DELETE RESTRICT,
    reel_alternative_id INTEGER NOT NULL,
    user_id             UUID NOT NULL CONSTRAINT fk_answers_user REFERENCES users(id),
    frames_revealed     SMALLINT NOT NULL CHECK (frames_revealed BETWEEN 1 AND 5),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, reel_id), -- one guess per reel

    -- chosen alternative must belong to the reel
    FOREIGN KEY (reel_alternative_id, reel_id) REFERENCES reel_alternatives(id, reel_id)
);

CREATE INDEX idx_answers_reel_id ON answers(reel_id);
CREATE INDEX idx_answers_reel_alternative_id ON answers(reel_alternative_id);

-- +goose Down

DROP TABLE answers;
DROP TABLE reel_frames;
DROP TABLE frames;
DROP TABLE reel_alternatives;
DROP TABLE reels;
DROP TABLE games;
