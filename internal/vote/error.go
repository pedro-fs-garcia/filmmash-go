package vote

import "errors"

var ErrDuplicateEntry = errors.New("vote already exists")
var ErrFilmNotInDuel = errors.New("film does not belong to duel")
