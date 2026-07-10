-- name: InsertDuel :one
INSERT INTO duels (film_a_id, film_b_id)
VALUES ($1, $2)
RETURNING id;

-- name: GetDuelFilms :many
SELECT films.id, films.title, films.release_year, films.image_path, films.rating,
    directors.id AS director_id, directors.name AS director_name
FROM films
JOIN directors ON films.director_id = directors.id
WHERE films.id IN (
    SELECT film_a_id FROM duels WHERE duels.id = $1
    UNION
    SELECT film_b_id FROM duels WHERE duels.id = $1
);

-- name: GetDuelRatingsForUpdate :many
SELECT id, rating
FROM films
WHERE films.id IN (
    SELECT film_a_id FROM duels WHERE duels.id = $1
    UNION
    SELECT film_b_id FROM duels WHERE duels.id = $1
)
ORDER BY films.id
FOR UPDATE;

-- name: SelectRandomFilms :many
SELECT films.id, films.title, films.release_year, films.image_path, films.rating,
    directors.id AS director_id, directors.name AS director_name
FROM films
JOIN directors ON films.director_id = directors.id
ORDER BY RANDOM() LIMIT 2;

-- name: ComposeDuel :one
WITH random_id AS (
    SELECT films.id FROM films
    WHERE films.id <> sqlc.arg(winner_id)
    ORDER BY RANDOM() LIMIT 1
),
inserted_duel AS (
    INSERT INTO duels (film_a_id, film_b_id)
    SELECT sqlc.arg(winner_id), random_id.id FROM random_id
    RETURNING duels.id AS duel_id, film_a_id, film_b_id
)
SELECT i.duel_id,
    fa.id AS film_a_id, fa.title AS film_a_title, fa.release_year AS film_a_release_year,
    fa.image_path AS film_a_image_path, fa.rating AS film_a_rating,
    da.id AS director_a_id, da.name AS director_a_name,
    fb.id AS film_b_id, fb.title AS film_b_title, fb.release_year AS film_b_release_year,
    fb.image_path AS film_b_image_path, fb.rating AS film_b_rating,
    db.id AS director_b_id, db.name AS director_b_name
FROM inserted_duel i
JOIN films fa ON fa.id = i.film_a_id JOIN directors da ON da.id = fa.director_id
JOIN films fb ON fb.id = i.film_b_id JOIN directors db ON db.id = fb.director_id;

-- name: CountPendingDuels :one
SELECT COUNT(*) FROM duels d
WHERE NOT EXISTS (SELECT 1 FROM votes v WHERE v.duel_id = d.id);
