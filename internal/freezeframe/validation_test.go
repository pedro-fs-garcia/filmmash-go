package freezeframe_test

import (
	"errors"
	"filmmash/internal/film"
	"filmmash/internal/freezeframe"
	"strings"
	"testing"
	"time"
)

func TestFrameValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid frame", func(t *testing.T) {
		f := BuildValidFrame(23, 4)
		if err := f.Validate(); err != nil {
			t.Fatalf("got %T, wanted no error", err)
		}
	})

	t.Run("invalid film_id", func(t *testing.T) {
		f := BuildValidFrame(123, 4)
		f.FilmID = 0
		err := f.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrRequired) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrRequired.Error())
		}
	})

	t.Run("invalid image_path", func(t *testing.T) {
		f := BuildValidFrame(123, 4)
		f.ImagePath = ""
		err := f.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrRequired) {
			t.Fatalf("wrong error type. got %T, want %T", err, freezeframe.ErrRequired.Error())
		}
	})
}

func TestAlternativeValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid alternative", func(t *testing.T) {
		f := BuildFilm(34)
		a := BuildValidAlternative(1, f)
		if err := a.Validate(); err != nil {
			t.Fatalf("got %T, wanted no error", err)
		}
	})

	t.Run("invalid film_id", func(t *testing.T) {
		f := BuildFilm(0)
		a := BuildValidAlternative(23, f)
		err := a.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrRequired) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrRequired.Error())
		}
	})

	t.Run("invalid seq", func(t *testing.T) {
		f := BuildFilm(34)
		a := BuildValidAlternative(9, f)
		err := a.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrOutOfRange) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrOutOfRange.Error())
		}
	})
}

func TestReelFrameValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid ReelFrame", func(t *testing.T) {
		rf := BuildValidReelFrame(32, 1)
		if err := rf.Validate(); err != nil {
			t.Errorf("got %T, wanted no error", err)
		}
	})

	t.Run("invalid seq", func(t *testing.T) {
		fr := BuildValidReelFrame(32, 1)
		fr.Seq = 6
		err := fr.Validate()
		if err == nil {
			t.Fatalf("got no error, wanted %T", freezeframe.ErrOutOfRange)
		}
		if !errors.Is(err, freezeframe.ErrOutOfRange) {
			t.Fatalf("wrong error type; got %T, wanted %T", err, freezeframe.ErrOutOfRange.Error())
		}
	})
}

func TestReelValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid reel", func(t *testing.T) {
		r := BuildValidReel(3)
		if err := r.Validate(); err != nil {
			t.Fatalf("got %v, wanted no error", err)
		}
	})

	t.Run("invalid seq", func(t *testing.T) {
		r := BuildValidReel(3)
		r.Seq = 9
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrOutOfRange) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrOutOfRange.Error())
		}
	})

	t.Run("missing film", func(t *testing.T) {
		r := BuildValidReel(3)
		r.Film = film.Film{}
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrRequired) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrRequired.Error())
		}
	})

	t.Run("wrong reel_frame count", func(t *testing.T) {
		r := BuildValidReel(3)
		r.ReelFrames = r.ReelFrames[:4]
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrWrongCount) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrWrongCount.Error())
		}
	})

	t.Run("reel_frames out of order", func(t *testing.T) {
		r := BuildValidReel(3)
		r.ReelFrames[0].Seq, r.ReelFrames[1].Seq = r.ReelFrames[1].Seq, r.ReelFrames[0].Seq
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrOutOfOrder) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrOutOfOrder.Error())
		}
	})

	t.Run("frame from another film", func(t *testing.T) {
		r := BuildValidReel(3)
		r.ReelFrames[2].Frame.FilmID = 99999
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrMismatch) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrMismatch.Error())
		}
	})

	t.Run("duplicate frame image", func(t *testing.T) {
		r := BuildValidReel(3)
		r.ReelFrames[1].Frame.ImagePath = r.ReelFrames[0].Frame.ImagePath
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrDuplicate) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrDuplicate.Error())
		}
	})

	t.Run("wrong alternative count", func(t *testing.T) {
		r := BuildValidReel(3)
		r.Alternatives = r.Alternatives[:3]
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrWrongCount) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrWrongCount.Error())
		}
	})

	t.Run("alternatives out of order", func(t *testing.T) {
		r := BuildValidReel(3)
		r.Alternatives[0].Seq, r.Alternatives[1].Seq = r.Alternatives[1].Seq, r.Alternatives[0].Seq
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrOutOfOrder) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrOutOfOrder.Error())
		}
	})

	t.Run("no right answer", func(t *testing.T) {
		r := BuildValidReel(3)
		r.Alternatives[0].Film = BuildFilm(9999)
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrWrongCount) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrWrongCount.Error())
		}
	})

	t.Run("duplicate alternative film", func(t *testing.T) {
		r := BuildValidReel(3)
		r.Alternatives[3].Film = r.Alternatives[2].Film
		err := r.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrDuplicate) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrDuplicate.Error())
		}
	})
}

func TestGameValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid game", func(t *testing.T) {
		g := BuildValidGame()
		if err := g.Validate(); err != nil {
			t.Fatalf("got %v, wanted no error", err)
		}
	})

	t.Run("missing valid_at", func(t *testing.T) {
		g := BuildValidGame()
		g.ValidAt = time.Time{}
		err := g.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrRequired) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrRequired.Error())
		}
	})

	t.Run("wrong reel count", func(t *testing.T) {
		g := BuildValidGame()
		g.Reels = g.Reels[:4]
		err := g.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrWrongCount) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrWrongCount.Error())
		}
	})

	t.Run("reels out of order", func(t *testing.T) {
		g := BuildValidGame()
		g.Reels[0].Seq, g.Reels[1].Seq = g.Reels[1].Seq, g.Reels[0].Seq
		err := g.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrOutOfOrder) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrOutOfOrder.Error())
		}
	})

	t.Run("duplicate reel film", func(t *testing.T) {
		g := BuildValidGame()
		g.Reels[1].Film = g.Reels[0].Film
		err := g.Validate()
		if err == nil {
			t.Fatalf("got no error")
		}
		if !errors.Is(err, freezeframe.ErrDuplicate) {
			t.Fatalf("wrong error type; got %T, want %T", err, freezeframe.ErrDuplicate.Error())
		}
	})
}

func TestErrorReportIsReadable(t *testing.T) {
	t.Parallel()

	g := BuildValidGame()
	g.ValidAt = time.Time{}
	g.Reels[0].ReelFrames[1].Frame.ImagePath = ""
	g.Reels[0].ReelFrames[2].Difficulty = 42
	g.Reels[1].Alternatives[0].Film = film.Film{Id: 77}
	g.Reels[2].Alternatives[3].Seq = 9
	g.Reels[3].Film = g.Reels[2].Film
	g.Reels = g.Reels[:4]

	err := g.Validate()
	if err == nil {
		t.Fatal("got no error, want a report")
	}
	t.Log(err.Error())
	for _, line := range strings.Split(err.Error(), "; ") {
		t.Log(line)
	}
}
