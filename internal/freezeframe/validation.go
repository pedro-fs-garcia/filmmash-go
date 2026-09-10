package freezeframe

import "fmt"

func (f *Frame) Validate() error {
	var errs []error
	if f.ImagePath == "" {
		errs = append(errs, &Violation{"image_path", ErrRequired, "must be set"})
	}
	if f.FilmID == 0 {
		errs = append(errs, &Violation{"film.id", ErrRequired, "must be set"})
	}
	if len(errs) == 0 {
		return nil
	}
	return group("", errs...)
}

func (a *Alternative) Validate() error {
	var errs []error
	if a.Seq < 1 || a.Seq > 4 {
		errs = append(errs, &Violation{"seq", ErrOutOfRange, fmt.Sprintf("must be between 1 and 4, got %d", a.Seq)})
	}
	if a.Film.Id == 0 {
		errs = append(errs, &Violation{"film.id", ErrRequired, "film.id must be set"})
	}
	if len(errs) == 0 {
		return nil
	}
	return group("", errs...)
}

func (rf *ReelFrame) Validate() error {
	var errs []error
	if rf.Seq > 5 || rf.Seq < 1 {
		errs = append(errs, &Violation{"seq", ErrOutOfRange, fmt.Sprintf("must be between 1 and 5, got %d", rf.Seq)})
	}
	if rf.Difficulty > 10 || rf.Difficulty < 1 {
		errs = append(errs, &Violation{
			"difficulty",
			ErrOutOfRange,
			fmt.Sprintf("must be between 1 and 10, got %d", rf.Difficulty),
		})
	}
	errs = append(errs, rf.Frame.Validate())
	return group("", errs...)
}

func (r *Reel) validateReelFrames() []error {
	var errs []error
	seenPaths := make(map[string]bool, len(r.ReelFrames))
	for i, rf := range r.ReelFrames {
		var e []error

		if rf.Seq != int16(i+1) {
			e = append(e, &Violation{
				"seq", ErrOutOfOrder,
				fmt.Sprintf("expected %d, got %d", i+1, rf.Seq),
			})
		}

		if rf.Frame.FilmID != int32(r.Film.Id) {
			e = append(e, &Violation{
				"frame.film_id", ErrMismatch,
				fmt.Sprintf("frame at seq %d belongs to film %d, want %d", rf.Seq, rf.Frame.FilmID, r.Film.Id),
			})
		}

		if seenPaths[rf.Frame.ImagePath] {
			e = append(e, &Violation{
				"frame.image_path", ErrDuplicate,
				fmt.Sprintf("duplicate frame image %s", rf.Frame.ImagePath),
			})
		}
		seenPaths[rf.Frame.ImagePath] = true
		e = append(e, rf.Validate())
		if ve := group(fmt.Sprintf("reel_frames[%d]", i+1), e...); ve != nil {
			errs = append(errs, ve)
		}
	}
	return errs
}

func (r *Reel) validateAlternatives() []error {
	var errs []error
	seenFilms := make(map[int]bool, len(r.Alternatives))
	rightAns := 0
	for i, a := range r.Alternatives {
		var e []error
		if a.Seq != int16(i+1) {
			e = append(e, &Violation{
				"seq", ErrOutOfOrder,
				fmt.Sprintf("expected %d, got %d", i+1, a.Seq),
			})
		}

		if a.Film.Id == r.Film.Id {
			rightAns += 1
		}
		if seenFilms[a.Film.Id] {
			e = append(e, &Violation{
				"film.id", ErrDuplicate,
				fmt.Sprintf("duplicate alternative for film %d", a.Film.Id),
			})
		}
		seenFilms[a.Film.Id] = true
		e = append(e, a.Validate())
		if ve := group(fmt.Sprintf("alternatives[%d]", i+1), e...); ve != nil {
			errs = append(errs, ve)
		}
	}
	if rightAns != 1 {
		errs = append(errs, &Violation{
			"alternatives", ErrWrongCount,
			fmt.Sprintf("reel must have exactly one right answer, got %d", rightAns),
		})
	}
	return errs
}

func (r *Reel) Validate() error {
	var errs []error
	if r.Seq < 1 || r.Seq > 5 {
		errs = append(errs, &Violation{"seq", ErrOutOfRange,
			fmt.Sprintf("must be between 1 and 5, got %d", r.Seq),
		})
	}
	if r.Film.Id == 0 {
		errs = append(errs, &Violation{"film.id", ErrRequired,
			fmt.Sprintf("film id must be set, got %d", r.Film.Id),
		})
	}

	if len(r.ReelFrames) != 5 {
		errs = append(errs, &Violation{"reel_frames", ErrWrongCount,
			fmt.Sprintf("must have exactly 5 frames, got %d", len(r.ReelFrames)),
		})
	}
	rfErrs := r.validateReelFrames()
	errs = append(errs, rfErrs...)

	if len(r.Alternatives) != 4 {
		errs = append(errs, &Violation{
			"alternatives",
			ErrWrongCount,
			fmt.Sprintf("must have exactly 4 alternatives, got %d", len(r.Alternatives)),
		})
	}
	altErrs := r.validateAlternatives()
	errs = append(errs, altErrs...)
	return group("", errs...)
}

func (g *Game) validateReels() []error {
	var errs []error

	if len(g.Reels) != 5 {
		errs = append(errs, &Violation{"reels", ErrWrongCount,
			fmt.Sprintf("must have exactly 5 reels, got %d", len(g.Reels)),
		})
	}

	seenFilms := make(map[int]bool, len(g.Reels))
	for i, r := range g.Reels {
		var e []error

		if r.Seq != int16(i+1) {
			e = append(e, &Violation{
				"seq", ErrOutOfOrder,
				fmt.Sprintf("expected %d, got %d", i+1, r.Seq),
			})
		}

		if seenFilms[r.Film.Id] {
			e = append(e, &Violation{
				"", ErrDuplicate,
				fmt.Sprintf("duplicate reel for film %d", r.Film.Id),
			})
		}
		seenFilms[r.Film.Id] = true
		e = append(e, r.Validate())
		if ve := group(fmt.Sprintf("reels[%d]", i+1), e...); ve != nil {
			errs = append(errs, ve)
		}
	}
	return errs
}

func (g *Game) Validate() error {
	var errs []error
	if g.ValidAt.IsZero() {
		errs = append(errs, &Violation{
			"valid_at", ErrRequired,
			fmt.Sprintf("valid_at date must be set, got '%s'", g.ValidAt.String()),
		})
	}
	reelErrs := g.validateReels()
	errs = append(errs, reelErrs...)

	ve := group("game", errs...)
	if ve == nil {
		return nil
	}
	return fmt.Errorf("invalid game: %w", ve)
}
