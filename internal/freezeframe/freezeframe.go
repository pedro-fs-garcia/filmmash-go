package freezeframe

import (
	"filmmash/internal/film"
	"time"

	"github.com/google/uuid"
)

type Game struct {
	ID      int32
	ValidAt time.Time
	Reels   []Reel
}

type Reel struct {
	ID  int32
	Seq int16

	Film         film.Film
	ReelFrames   []ReelFrame
	Alternatives []Alternative
}

type Frame struct {
	ID        int32
	FilmID    int32
	ImagePath string
}

type ReelFrame struct {
	ID         int32
	ImagePath  string
	Seq        int16
	Difficulty int16
}

type Alternative struct {
	ID   int32
	Seq  int16
	Film film.Film
}

type Answer struct {
	ID                int32
	ReelID            int32
	ReelAlternativeID int32
	UserID            uuid.UUID
	FramesRevealed    int16
	CreatedAt         time.Time
}

type Round struct {
	ReelSeq, ReelTotal int16
	Frames             []ReelFrame
	Alternatives       []Alternative
	Result             *Result
}

type Result struct {
	ChosenID, CorrectID int32
	Correct             bool
	Points              int
}

func (g *Game) Validate() error {
	return nil
}
