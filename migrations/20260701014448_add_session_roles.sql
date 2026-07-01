-- +goose Up
SELECT 'up SQL query';

ALTER TABLE sessions ADD COLUMN roles TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
SELECT 'down SQL query';

ALTER TABLE sessions DROP COLUMN roles;
