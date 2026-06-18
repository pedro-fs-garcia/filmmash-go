-- +goose Up
SELECT 'up SQL query';

CREATE INDEX idx_films_rating_id ON films(rating, id);


-- +goose Down
SELECT 'down SQL query';

DROP INDEX idx_films_rating_id;
