package freezeframe_test

import (
	"filmmash/internal/film"
	"filmmash/internal/freezeframe"
	"fmt"
	"time"
)

func BuildFilm(id int) film.Film {
	return film.Film{Id: id, Title: fmt.Sprintf("Film %d", id)}
}

func BuildValidFrame(filmID int32, seq int16) freezeframe.Frame {
	return freezeframe.Frame{
		FilmID:    filmID,
		ImagePath: fmt.Sprintf("film%d/frame%d.jpg", filmID, seq),
	}
}

func BuildValidReelFrame(filmID int32, seq int16) freezeframe.ReelFrame {
	return freezeframe.ReelFrame{
		Seq:        seq,
		Difficulty: seq * 2,
		Frame:      BuildValidFrame(filmID, seq),
	}
}

func BuildValidAlternative(seq int16, f film.Film) freezeframe.Alternative {
	return freezeframe.Alternative{
		Seq:  seq,
		Film: f,
	}
}

func BuildValidReel(seq int16) freezeframe.Reel {
	f := BuildFilm(int(seq) * 10)
	reel := freezeframe.Reel{
		Seq:  seq,
		Film: f,
	}

	for i := int16(1); i <= 5; i++ {
		reel.ReelFrames = append(reel.ReelFrames, BuildValidReelFrame(int32(f.Id), i))
	}

	reel.Alternatives = append(reel.Alternatives, BuildValidAlternative(1, f))
	for i := int16(2); i <= 4; i++ {
		reel.Alternatives = append(reel.Alternatives, BuildValidAlternative(i, BuildFilm(f.Id+int(i))))
	}

	return reel
}

func BuildValidGame() freezeframe.Game {
	g := freezeframe.Game{
		ValidAt: time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
	}

	for i := int16(1); i <= 5; i++ {
		g.Reels = append(g.Reels, BuildValidReel(i))
	}

	return g
}
