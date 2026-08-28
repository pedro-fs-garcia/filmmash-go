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
RETURNING id;

-- name: InsertReelFrames :batchexec
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
