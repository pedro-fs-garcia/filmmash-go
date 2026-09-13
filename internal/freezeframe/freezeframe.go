package freezeframe

import (
	"filmmash/internal/film"
	"time"

	"github.com/google/uuid"
)

type Game struct {
	ID      int32     `json:"id"`
	ValidAt time.Time `json:"valid_at"`
	Reels   []Reel    `json:"reels"`
}

type Reel struct {
	ID  int32 `json:"id"`
	Seq int16 `json:"seq"`

	Film         film.Film     `json:"film"`
	ReelFrames   []ReelFrame   `json:"reel_frames"`
	Alternatives []Alternative `json:"alternatives"`
}

type Frame struct {
	ID        int32  `json:"id"`
	FilmID    int32  `json:"film_id"`
	ImagePath string `json:"image_path"`
}

type ReelFrame struct {
	ID         int32 `json:"id"`
	Seq        int16 `json:"seq"`
	Difficulty int16 `json:"difficulty"`
	Frame      Frame `json:"frame"`
}

type Alternative struct {
	ID   int32     `json:"id"`
	Seq  int16     `json:"seq"`
	Film film.Film `json:"film"`
}

type Answer struct {
	ID                int32     `json:"id"`
	ReelID            int32     `json:"reel_id"`
	ReelAlternativeID int32     `json:"reel_alternative_id"`
	UserID            uuid.UUID `json:"user_id"`
	FramesRevealed    int16     `json:"frames_revealed"`
	CreatedAt         time.Time `json:"created_at"`
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
