-- +goose Up
SELECT 'up SQL query';

CREATE INDEX idx_votes_user_id ON votes(user_id);

-- +goose Down
SELECT 'down SQL query';

DROP INDEX idx_votes_user_id;