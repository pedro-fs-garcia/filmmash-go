package freezeframe_test

import (
	"filmmash/internal/film"
	"filmmash/internal/freezeframe"
	"fmt"
	"time"
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
