-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    zitadel_sub TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE votes ADD COLUMN user_id UUID REFERENCES users(id);


-- +goose Down
ALTER TABLE votes DROP COLUMN user_id;
DROP TABLE users;
