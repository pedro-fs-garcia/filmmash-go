-- +goose Up
SELECT 'up SQL query';

CREATE TABLE session_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    event TEXT NOT NULL,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_events_user_id ON session_events(user_id);
CREATE INDEX idx_session_events_session_id_user_id ON session_events(session_id, user_id);


-- +goose Down
SELECT 'down SQL query';

DROP INDEX idx_session_events_session_id_user_id
DROP INDEX idx_session_events_user_id
