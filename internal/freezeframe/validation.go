package freezeframe

import (
	"fmt"
	"strings"
)

type Violation struct {
	Where   string
	Message string
}

func (v Violation) Error() string {
	return v.Where + ": " + v.Message
}

type ValidationErrors []Violation

func (ve ValidationErrors) Error() string {
	msgs := make([]string, len(ve))
	for i, v := range ve {
		msgs[i] = v.Error()
	}
	return strings.Join(msgs, "; ")
}

func (ve ValidationErrors) Unwrap() []error {
	errs := make([]error, len(ve))
	for i, v := range ve {
		errs[i] = v
	}
	return errs
}

func prefixed(path string, errs ValidationErrors) ValidationErrors {
	out := make(ValidationErrors, len(errs))
	for i, e := range errs {
		where := path
		if e.Where != "" {
			where += "." + e.Where
		}
		out[i] = Violation{where, e.Message}
	}
	return out
}

func (f *Frame) Validate() ValidationErrors {
	var ve ValidationErrors
	if f.ImagePath == "" {
		ve = append(ve, Violation{"image_path", "must be set"})
	}
	if f.FilmID == 0 {
		ve = append(ve, Violation{"film.id", "must be set"})
	}
	if len(ve) == 0 {
		return nil
	}
	return ve
}

func (a *Alternative) Validate() ValidationErrors {
	var ve ValidationErrors
	if a.Seq < 1 || a.Seq > 4 {
		ve = append(ve, Violation{"seq", fmt.Sprintf("must be between 1 and 4, got %d", a.Seq)})
	}
	if a.Film.Id == 0 {
		ve = append(ve, Violation{"film.id", "must be set"})
	}
	if len(ve) == 0 {
		return nil
	}
	return ve
}

func (rf *ReelFrame) Validate() ValidationErrors {
	var ve ValidationErrors

	if rf.Seq > 5 || rf.Seq < 1 {
		ve = append(ve, Violation{"seq", fmt.Sprintf("seq attribute must be between 1 and 5, got %d", rf.Seq)})
	}
	if rf.Difficulty > 10 || rf.Difficulty < 1 {
		ve = append(ve, Violation{
			"difficulty",
			fmt.Sprintf("difficulty must be between 1 and 10, got %d", rf.Difficulty),
		})
	}

	frameErrs := prefixed("frame", rf.Frame.Validate())
	ve = append(ve, frameErrs...)

	if len(ve) == 0 {
		return nil
	}
	return ve
}

func (r *Reel) validateReelFrames() ValidationErrors {
	var ve ValidationErrors

	seenPaths := make(map[string]bool, len(r.ReelFrames))
	for i, rf := range r.ReelFrames {
		path := fmt.Sprintf("reel_frames[%d]", rf.Seq)

		if rf.Seq != int16(i+1) {
			ve = append(ve, Violation{
				path + ".seq",
				fmt.Sprintf("expected seq = %d, got %d", i+1, rf.Seq),
			})
		}

		if rf.Frame.FilmID != int32(r.Film.Id) {
			ve = append(ve, Violation{
				path + ".frame.film_id",
				fmt.Sprintf("frame at seq %d belongs to film %d, want %d", rf.Seq, rf.Frame.FilmID, r.Film.Id),
			})
		}

		if seenPaths[rf.Frame.ImagePath] {
			ve = append(ve, Violation{
				path + ".frame.image_path",
				fmt.Sprintf("duplicate frame image %s", rf.Frame.ImagePath),
			})
		}
		seenPaths[rf.Frame.ImagePath] = true

		rfErrors := prefixed(path, rf.Validate())
		ve = append(ve, rfErrors...)
	}
	if len(ve) == 0 {
		return nil
	}
	return ve
}

func (r *Reel) validateAlternatives() ValidationErrors {
	var ve ValidationErrors

	seenFilms := make(map[int]bool, len(r.Alternatives))
	rightAns := 0
	for i, a := range r.Alternatives {
		path := fmt.Sprintf("alternatives[%d]", a.Seq)

		if a.Seq != int16(i+1) {
			ve = append(ve, Violation{
				path + ".seq",
				fmt.Sprintf("expected seq = %d, got %d", i+1, a.Seq),
			})
		}

		if a.Film.Id == r.Film.Id {
			rightAns += 1
		}
		if seenFilms[a.Film.Id] {
			ve = append(ve, Violation{
				path + ".film.id",
				fmt.Sprintf("duplicate alternative for film %d", a.Film.Id),
			})
		}
		seenFilms[a.Film.Id] = true

		altErrs := prefixed(path, a.Validate())
		ve = append(ve, altErrs...)
	}
	if rightAns != 1 {
		ve = append(ve, Violation{
			"alternatives",
			fmt.Sprintf("reel must have exactly one right answer, got %d", rightAns),
		})
	}

	if len(ve) == 0 {
		return nil
	}
	return ve
}

func (r *Reel) Validate() ValidationErrors {
	var ve ValidationErrors
	if r.Seq < 1 || r.Seq > 5 {
		ve = append(ve, Violation{"seq", fmt.Sprintf("seq attribute must be between 1 and 5, got %d", r.Seq)})
	}
	if r.Film.Id == 0 {
		ve = append(ve, Violation{"film.id", fmt.Sprintf("film id must be set, got %d", r.Film.Id)})
	}

	if len(r.ReelFrames) != 5 {
		ve = append(ve, Violation{"reel_frames", fmt.Sprintf("must have exactly 5 frames, got %d", len(r.ReelFrames))})
	}
	rfErrs := r.validateReelFrames()
	ve = append(ve, rfErrs...)

	if len(r.Alternatives) != 4 {
		ve = append(ve, Violation{
			"alternatives",
			fmt.Sprintf("must have exactly 4 alternatives, got %d", len(r.Alternatives)),
		})
	}
	altErrs := r.validateAlternatives()
	ve = append(ve, altErrs...)

	if len(ve) == 0 {
		return nil
	}
	return ve
}

func (g *Game) validateReels() ValidationErrors {
	var ve ValidationErrors

	if len(g.Reels) != 5 {
		ve = append(ve, Violation{"reels", fmt.Sprintf("must have exactly 5 reels, got %d", len(g.Reels))})
	}

	seenFilms := make(map[int]bool, len(g.Reels))
	for i, r := range g.Reels {
		path := fmt.Sprintf("reels[%d]", r.Seq)

		if r.Seq != int16(i+1) {
			ve = append(ve, Violation{
				path + ".seq",
				fmt.Sprintf("expected %d, got %d", i+1, r.Seq),
			})
		}

		if seenFilms[r.Film.Id] {
			ve = append(ve, Violation{
				path,
				fmt.Sprintf("duplicate reel for film %d", r.Film.Id),
			})
		}
		seenFilms[r.Film.Id] = true

		reelErrs := prefixed(path, r.Validate())
		ve = append(ve, reelErrs...)
	}
	if len(ve) == 0 {
		return nil
	}
	return ve
}

func (g *Game) Validate() error {
	var ve ValidationErrors
	if g.ValidAt.IsZero() {
		ve = append(ve, Violation{
			"game.valid_at",
			fmt.Sprintf("valid_at date must be set, got '%s'", g.ValidAt.String()),
		})
	}
	reelErrs := prefixed("game", g.validateReels())
	ve = append(ve, reelErrs...)

	if len(ve) == 0 {
		return nil
	}
	return fmt.Errorf("invalid game: %w", ve)
}
