-- +goose Up
SELECT 'up SQL query';

TRUNCATE sessions;
ALTER TABLE sessions ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
ALTER TABLE sessions ADD CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE session_events RENAME COLUMN user_id TO zitadel_sub;
ALTER INDEX idx_session_events_user_id RENAME TO idx_session_events_zitadel_sub;
ALTER INDEX idx_session_events_session_id_user_id RENAME TO idx_session_events_session_id_zitadel_sub;


-- +goose Down
SELECT 'down SQL query';

ALTER INDEX idx_session_events_session_id_zitadel_sub RENAME TO idx_session_events_session_id_user_id;
ALTER INDEX idx_session_events_zitadel_sub RENAME TO idx_session_events_user_id;
ALTER TABLE session_events RENAME COLUMN zitadel_sub TO user_id;
ALTER TABLE sessions DROP CONSTRAINT fk_sessions_user;
ALTER TABLE sessions ALTER COLUMN user_id TYPE TEXT USING user_id::text;
