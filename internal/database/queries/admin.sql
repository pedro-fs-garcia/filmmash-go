-- name: ListUsersWithVotes :many
SELECT u.id, u.idp_sub, u.created_at, COUNT(*) AS votes
FROM users u
JOIN votes v ON u.id = v.user_id
WHERE u.id > $1
GROUP BY u.id, u.idp_sub, u.created_at
ORDER BY u.id
LIMIT $2;
