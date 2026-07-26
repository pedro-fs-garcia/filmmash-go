package film

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("resource not found")
var ErrDuplicateEntry = errors.New("register already exists")
var ErrInvalidInput = errors.New("invalid input value")
var ErrInvalidRating = errors.New("invalid film rating value")
var ErrMissingData = errors.New("missing required field")

func parseDBError(ctxMsg string, err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", ctxMsg, ErrNotFound)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%s: %w [internal: %w]", ctxMsg, ErrDuplicateEntry, err)
		case "23514":
			return fmt.Errorf("%s: %w [internal]: %w", ctxMsg, ErrInvalidInput, err)
		case "23502":
			return fmt.Errorf(
				"%s (missing field '%s'): %w [internal: %w]",
				ctxMsg, pgErr.ColumnName, ErrMissingData, err,
			)
		}
	}
	return fmt.Errorf("%s: %w", ctxMsg, err)
}
