package freezeframe

import (
	"cmp"
	"errors"
	"filmmash/internal/film"
	"fmt"
	"slices"
	"strings"
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
	Seq        int16
	Difficulty int16
	Frame      Frame
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

func (f *Frame) Validate() error {
	msg := []string{}
	if f.ImagePath == "" {
		msg = append(msg, "image_path must be set")
	}
	if f.FilmID == 0 {
		msg = append(msg, "films must be set")
	}
	if len(msg) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrInvalidFrame, strings.Join(msg, "; "))
}

func (a *Alternative) Validate() error {
	msg := []string{}
	if a.Seq < 1 || a.Seq > 4 {
		msg = append(msg, "seq must be between 1 and 4")
	}
	if a.Film.Id == 0 {
		msg = append(msg, "film must be set")
	}
	if len(msg) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrInvalidAlternative, strings.Join(msg, "; "))
}

func (rf *ReelFrame) Validate() error {
	var errs []error
	if rf.Seq > 5 || rf.Seq < 1 {
		errs = append(errs, errors.New("seq attribute must be between 1 and 5"))
	}
	if rf.Difficulty > 10 || rf.Difficulty < 1 {
		errs = append(errs, errors.New("difficulty must be between 1 and 10"))
	}
	if err := rf.Frame.Validate(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrInvalidReelFrame, errors.Join(errs...))
}

func (r *Reel) Validate() error {
	var errs []error
	if r.Seq < 1 || r.Seq > 5 {
		errs = append(errs, fmt.Errorf("seq attribute must be between 1 and 5, got %d", r.Seq))
	}
	if r.Film.Id == 0 {
		errs = append(errs, ErrFilmMustBeSet)
	}

	if len(r.ReelFrames) != 5 {
		errs = append(errs, fmt.Errorf("reel must have exactly 5 frames, got %d", len(r.ReelFrames)))
	}
	if len(r.Alternatives) != 4 {
		errs = append(errs, fmt.Errorf("reel must have exactly 4 alternatives, got %d", len(r.Alternatives)))
	}

	reelFrames := slices.Clone(r.ReelFrames)
	slices.SortFunc(reelFrames, func(a, b ReelFrame) int {
		return cmp.Compare(a.Seq, b.Seq)
	})

	seenPaths := make(map[string]bool, len(reelFrames))
	for i, f := range reelFrames {
		if err := f.Validate(); err != nil {
			errs = append(errs, err)
			continue
		}
		if f.Seq != int16(i+1) {
			errs = append(errs, errors.New("sequence of reelframes must be incremental in 1"))
		}

		if f.Frame.FilmID != int32(r.Film.Id) {
			errs = append(errs, fmt.Errorf(
				"frame at seq %d belongs to film %d, want %d", f.Seq, f.Frame.FilmID, r.Film.Id))
		}

		if seenPaths[f.Frame.ImagePath] {
			errs = append(errs, fmt.Errorf("duplicate frame image %s", f.Frame.ImagePath))
		}
		seenPaths[f.Frame.ImagePath] = true
	}

	alts := slices.Clone(r.Alternatives)
	slices.SortFunc(alts, func(a, b Alternative) int {
		return cmp.Compare(a.Seq, b.Seq)
	})

	seenFilms := make(map[int]bool, len(alts))
	rightAns := 0
	for i, a := range alts {
		if err := a.Validate(); err != nil {
			errs = append(errs, err)
			continue
		}
		if a.Seq != int16(i+1) {
			errs = append(errs, errors.New("sequence of alternatives must be incremental in 1"))
		}
		if a.Film.Id == r.Film.Id {
			rightAns += 1
		}
		if seenFilms[a.Film.Id] {
			errs = append(errs, fmt.Errorf("duplicate alternative film %d", a.Film.Id))
		}
		seenFilms[a.Film.Id] = true
	}
	if rightAns != 1 {
		errs = append(errs, fmt.Errorf("reel must have exactly one right answer, got %d", rightAns))
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrInvalidReel, errors.Join(errs...))
}

func (g *Game) Validate() error {
	return nil
}
