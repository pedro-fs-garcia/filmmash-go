-- +goose Up
ALTER TABLE games ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
-- Preserve legacy rows and their references without publishing incompatible games.
UPDATE games g SET active=FALSE WHERE
 (SELECT COUNT(*) FROM reels r WHERE r.game_id=g.id) <> 5 OR
 EXISTS(SELECT 1 FROM reels r WHERE r.game_id=g.id AND (
  (SELECT COUNT(*) FROM reel_frames rf WHERE rf.reel_id=r.id) <> 4 OR
  (SELECT COUNT(*) FROM reel_alternatives a WHERE a.reel_id=r.id) <> 5));
CREATE UNIQUE INDEX games_active_day ON games(valid_at) WHERE active;

-- +goose Down
DROP INDEX games_active_day;
ALTER TABLE games DROP COLUMN active;
