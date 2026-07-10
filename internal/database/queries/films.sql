-- name: InsertFilm :one
WITH new_director AS (
    INSERT INTO directors (id, name)
    VALUES (sqlc.arg(director_id), sqlc.arg(director_name))
    ON CONFLICT DO NOTHING
    RETURNING id AS director_id
),
director AS (
    SELECT director_id FROM new_director
    UNION ALL
    SELECT id AS director_id FROM directors WHERE id = sqlc.arg(director_id)
)
INSERT INTO films (id, title, release_year, director_id, image_path, rating)
SELECT sqlc.arg(id), sqlc.arg(title), sqlc.arg(release_year), director_id, sqlc.arg(image_path), sqlc.arg(rating) FROM director
RETURNING id;

-- name: InsertDirector :one
INSERT INTO directors (id, name)
VALUES ($1, $2)
RETURNING id;

-- name: GetDirector :one
SELECT id, name FROM directors WHERE id = $1;

-- name: GetFilm :one
SELECT films.id, films.title, films.release_year, films.image_path, films.rating,
    directors.id AS director_id, directors.name AS director_name
FROM films
JOIN directors ON films.director_id = directors.id
WHERE films.id = $1;

-- name: GetRandomFilm :one
SELECT films.id, films.title, films.release_year, films.image_path, films.rating,
    directors.id AS director_id, directors.name AS director_name
FROM films
JOIN directors ON films.director_id = directors.id
ORDER BY RANDOM() LIMIT 1;

-- name: ListFilmsByRating :many
SELECT films.id, films.title, films.release_year, films.image_path, films.rating,
    directors.id AS director_id, directors.name AS director_name
FROM films
JOIN directors ON films.director_id = directors.id
WHERE (
    sqlc.narg(last_seen_id)::int IS NULL
    OR (films.rating, films.id) < (sqlc.narg(last_seen_rating)::float, sqlc.narg(last_seen_id)::int)
)
ORDER BY films.rating DESC, films.id DESC
LIMIT $1;

-- name: SearchFilmsByTitle :many
SELECT films.id, films.title, films.release_year, films.image_path, films.rating,
    directors.id AS director_id, directors.name AS director_name
FROM films
JOIN directors ON films.director_id = directors.id
WHERE films.title ILIKE '%' || sqlc.arg(search)::text || '%';

-- name: UpdateFilmRating :execrows
UPDATE films SET rating = $1 WHERE id = $2;

-- name: UpdateFilmRatings :batchone
UPDATE films SET rating = $1 WHERE id = $2 RETURNING id;

-- name: CountFilms :one
SELECT COUNT(*) FROM films;
