package duel

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("Duel not found")
var ErrDuplicateEntry = errors.New("Duel already exists")
var ErrInvalidDuel = errors.New("Invalid duel")
var ErrMissingData = errors.New("Missing required data")
var ErrNotEnoughFilms = errors.New("Not enough films to create duel")
var ErrNoCandidates = fmt.Errorf("no candidates were found for duel: %w", ErrNotEnoughFilms)

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
		case "23503":
			return fmt.Errorf("%s: %w [internal]: %w", ctxMsg, ErrInvalidDuel, err)
		case "23505":
			return fmt.Errorf("%s: %w [internal: %w]", ctxMsg, ErrDuplicateEntry, err)
		case "23514":
			return fmt.Errorf("%s: %w [internal]: %w", ctxMsg, ErrInvalidDuel, err)
		case "23502":
			return fmt.Errorf(
				"%s (missing field '%s'): %w [internal: %w]",
				ctxMsg, pgErr.ColumnName, ErrMissingData, err,
			)
		}
	}
	return fmt.Errorf("%s: %w", ctxMsg, err)
}
