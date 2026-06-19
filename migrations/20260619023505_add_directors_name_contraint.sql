-- +goose Up
SELECT 'up SQL query';

ALTER TABLE directors DROP CONSTRAINT directors_name_check;
ALTER TABLE directors ADD CONSTRAINT directors_name_check CHECK (length(name) BETWEEN 2 AND 127);

-- +goose Down
SELECT 'down SQL query';

ALTER TABLE directors DROP CONSTRAINT directors_name_check;
ALTER TABLE directors ADD CONSTRAINT directors_name_check CHECK (length(name) <= 127);
