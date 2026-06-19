-- +goose Up
SELECT 'up SQL query';

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON films
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- +goose Down
SELECT 'down SQL query';

DROP TRIGGER IF EXISTS set_timestamp ON films;
DROP FUNCTION trigger_set_timestamp();