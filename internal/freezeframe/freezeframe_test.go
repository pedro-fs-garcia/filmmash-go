package freezeframe_test

import (
	"context"
	"filmmash/internal/film"
	"filmmash/internal/freezeframe"
	"fmt"
	"testing"
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

func validateIDs(t *testing.T, IDs []int32, expectLen int, source string) {
	t.Helper()
	if len(IDs) != expectLen {
		t.Fatalf("got %d %s IDs, wanted %d", len(IDs), source, expectLen)
	}
	for i, id := range IDs {
		if id == 0 {
			t.Fatalf("invalid id for %s[%d]; got %d", source, i, id)
		}
	}
}

func gameFilms(g freezeframe.Game) []film.Film {
	seen := make(map[int]bool)
	var films []film.Film
	add := func(f film.Film) {
		if !seen[f.Id] {
			seen[f.Id] = true
			films = append(films, f)
		}
	}
	for _, reel := range g.Reels {
		add(reel.Film)
		for _, alt := range reel.Alternatives {
			add(alt.Film)
		}
	}
	return films
}

func seedFilms(t *testing.T, ctx context.Context, films []film.Film) {
	t.Helper()
	repo := film.NewRepository(testPool)
	for _, f := range films {
		f.Year = 2000
		f.Director = film.Director{Id: f.Id, Name: fmt.Sprintf("Director %d", f.Id)}
		if err := repo.InsertFilm(ctx, &f); err != nil {
			t.Fatalf("seeding film %d: %v", f.Id, err)
		}
	}
}
