-- name: InsertGame :one
INSERT INTO games (valid_at)
VALUES (sqlc.arg(valid_at))
RETURNING id;

-- name: InsertReels :batchone
INSERT INTO reels (game_id, film_id, seq)
VALUES (sqlc.arg(game_id), sqlc.arg(film_id), sqlc.arg(seq))
RETURNING id;

-- name: InsertFrames :batchone
INSERT INTO frames (film_id, image_path)
VALUES (sqlc.arg(film_id), sqlc.arg(image_path))
ON CONFLICT (image_path) DO UPDATE SET image_path = EXCLUDED.image_path
RETURNING id;

-- name: InsertReelFrames :batchone
INSERT INTO reel_frames (reel_id, frame_id, difficulty, seq)
VALUES (sqlc.arg(reel_id), sqlc.arg(frame_id), sqlc.arg(difficulty), sqlc.arg(seq))
RETURNING id;

-- name: InsertReelAlternatives :batchone
INSERT INTO reel_alternatives (reel_id, film_id, seq)
VALUES (sqlc.arg(reel_id), sqlc.arg(film_id), sqlc.arg(seq))
RETURNING id;

-- name: InsertAnswer :one
INSERT INTO answers(reel_id, reel_alternative_id, user_id, frames_revealed)
VALUES (sqlc.arg(reel_id), sqlc.arg(reel_alternative_id), sqlc.arg(user_id), sqlc.arg(frames_revealed))
RETURNING id;

-- name: GetGame :one
SELECT id, valid_at FROM games WHERE id = sqlc.arg(game_id);

-- name: GetGamesIdByDate :many
SELECT id FROM games WHERE valid_at = sqlc.arg(valid_at);

-- name: GetGameReels :many
SELECT r.id, r.seq, f.id AS film_id, f.title AS film_title, f.release_year AS film_year
FROM reels r
JOIN films f ON f.id = r.film_id
WHERE r.game_id = sqlc.arg(game_id)
ORDER BY r.seq;

-- name: GetReelAlternatives :many
SELECT a.id, a.reel_id, a.seq, f.id AS film_id, f.title AS film_title, f.release_year AS film_year
FROM reel_alternatives a
JOIN reels r ON r.id = a.reel_id
JOIN films f ON f.id = a.film_id
WHERE r.game_id = sqlc.arg(game_id)
ORDER BY a.reel_id, a.seq;

-- name: GetReelFrames :many
SELECT rf.reel_id AS reel_id, rf.id, rf.difficulty, rf.seq, f.id AS frame_id, f.image_path AS image_path
FROM reel_frames rf
JOIN frames f ON f.id = rf.frame_id
JOIN reels r ON r.id = rf.reel_id
WHERE r.game_id = sqlc.arg(game_id)
ORDER BY rf.reel_id, rf.seq;
