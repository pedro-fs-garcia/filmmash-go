package duel

import (
	"filmmash/internal/film"
	"time"

	"github.com/google/uuid"
)

type Duel struct {
	Id    uuid.UUID
	FilmA *film.Film
	FilmB *film.Film
}

type Candidate struct {
	Id         int32
	Popularity float64
	DuelCount  int32
	CreatedAt  time.Time
}
