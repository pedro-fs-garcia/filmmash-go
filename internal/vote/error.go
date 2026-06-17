package vote

import "errors"

var ErrDuplicateEntry = errors.New("Vote already exists")
var ErrFilmNotInDuel = errors.New("Film does not belong to duel")
