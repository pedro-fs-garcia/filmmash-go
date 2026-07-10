-- name: InsertVote :one
INSERT INTO votes (duel_id, user_id, winner_id, loser_id, winner_rating_after, loser_rating_after)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: ListFilmVotes :many
SELECT v.winner_id, v.loser_id, v.winner_rating_after, v.loser_rating_after, v.created_at,
    fw.title AS winner_title, fw.release_year AS winner_release_year,
    fl.title AS loser_title, fl.release_year AS loser_release_year
FROM votes v
JOIN films fw ON fw.id = v.winner_id
JOIN films fl ON fl.id = v.loser_id
WHERE v.winner_id = $1 OR v.loser_id = $1
ORDER BY v.created_at DESC;

-- name: ListUserVotes :many
SELECT v.winner_id, v.loser_id, v.winner_rating_after, v.loser_rating_after, v.created_at,
    fw.title AS winner_title, fw.release_year AS winner_release_year, fw.image_path AS winner_image_path,
    fl.title AS loser_title, fl.release_year AS loser_release_year, fl.image_path AS loser_image_path
FROM votes v
JOIN films fw ON fw.id = v.winner_id
JOIN films fl ON fl.id = v.loser_id
WHERE v.user_id = $1
ORDER BY v.created_at DESC;

-- name: CountVotes :one
SELECT COUNT(*) FROM votes;
