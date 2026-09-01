package freezeframe_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"filmmash/internal/film"
	"filmmash/internal/freezeframe"
)

func buildFilm(id int) film.Film {
	return film.Film{Id: id, Title: fmt.Sprintf("Film %d", id)}
}

func buildValidFrame(filmID int32, seq int16) freezeframe.Frame {
	return freezeframe.Frame{
		FilmID:    filmID,
		ImagePath: fmt.Sprintf("film%d/frame%d.jpg", filmID, seq),
	}
}

func buildValidReelFrame(filmID int32, seq int16) freezeframe.ReelFrame {
	return freezeframe.ReelFrame{
		Seq:        seq,
		Difficulty: seq * 2,
		Frame:      buildValidFrame(filmID, seq),
	}
}

func buildValidAlternative(seq int16, f film.Film) freezeframe.Alternative {
	return freezeframe.Alternative{
		Seq:  seq,
		Film: f,
	}
}

func buildValidReel(seq int16) freezeframe.Reel {
	f := buildFilm(int(seq) * 10)
	reel := freezeframe.Reel{
		Seq:  seq,
		Film: f,
	}

	for i := int16(1); i <= 5; i++ {
		reel.ReelFrames = append(reel.ReelFrames, buildValidReelFrame(int32(f.Id), i))
	}

	reel.Alternatives = append(reel.Alternatives, buildValidAlternative(1, f))
	for i := int16(2); i <= 4; i++ {
		reel.Alternatives = append(reel.Alternatives, buildValidAlternative(i, buildFilm(f.Id+int(i))))
	}

	return reel
}

func buildValidGame() freezeframe.Game {
	g := freezeframe.Game{
		ValidAt: time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
	}

	for i := int16(1); i <= 5; i++ {
		g.Reels = append(g.Reels, buildValidReel(i))
	}

	return g
}

func TestErrorReportIsReadable(t *testing.T) {
	t.Parallel()

	g := buildValidGame()
	g.ValidAt = time.Time{}
	g.Reels[0].ReelFrames[1].Frame.ImagePath = ""       // frame problem
	g.Reels[0].ReelFrames[2].Difficulty = 42            // reel frame problem
	g.Reels[1].Alternatives[0].Film = film.Film{Id: 77} // reel loses its right answer
	g.Reels[2].Alternatives[3].Seq = 9                  // alternative problem
	g.Reels[3].Film = g.Reels[2].Film                   // duplicate film across reels
	g.Reels = g.Reels[:4]                               // and one reel short

	err := g.Validate()
	if err == nil {
		t.Fatal("got no error, want a report")
	}
	errs := strings.Split(err.Error(), ";")
	t.Logf("\n%v", errs)
}
