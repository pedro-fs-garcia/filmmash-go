package duel

import "errors"

var ErrNotFound = errors.New("Duel not found")
var ErrDuplicateEntry = errors.New("Duel already exists")
var ErrInvalidDuel = errors.New("Invalid duel")
var ErrNotEnoughFilms = errors.New("Not enough films to create duel")
