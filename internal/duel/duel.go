package duel

import (
	"filmmash/internal/film"

	"github.com/google/uuid"
)

type Duel struct {
	Id    uuid.UUID
	FilmA *film.Film
	FilmB *film.Film
}
