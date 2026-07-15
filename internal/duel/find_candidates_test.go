package duel_test

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"testing"

	"github.com/jackc/pgx/v5"
)

const testWindow float64 = 200

func candidateIds(cs []duel.Candidate) map[int32]bool {
	ids := make(map[int32]bool, len(cs))
	for _, c := range cs {
		ids[c.Id] = true
	}
	return ids
}

func assertCandidates(t *testing.T, got []duel.Candidate, want ...int) {
	t.Helper()
	ids := candidateIds(got)
	if len(got) != len(want) {
		t.Fatalf("got %d candidates %v, want %d: %v", len(got), ids, len(want), want)
	}
	for _, id := range want {
		if !ids[int32(id)] {
			t.Errorf("candidates %v are missing film %d", ids, id)
		}
	}
}

func TestFindCandidates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := duel.NewRepository(testPool)
	filmRepo := film.NewRepository(testPool)

	newTx := func(t *testing.T) (context.Context, pgx.Tx) {
		t.Helper()
		tx, err := testPool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		t.Cleanup(func() { tx.Rollback(ctx) })
		return database.InjectTx(ctx, tx), tx
	}

	seed := func(t *testing.T, txCtx context.Context, dir film.Director, films ...*film.Film) {
		t.Helper()
		for _, f := range films {
			f.Director = dir
			f.ImagePath = "img.jpg"
			if err := filmRepo.InsertFilm(txCtx, f); err != nil {
				t.Fatalf("seeding film %d: %v", f.Id, err)
			}
		}
	}

	t.Run("returns in-window films and excludes the winner", func(t *testing.T) {
		txCtx, _ := newTx(t)
		dir := film.Director{Id: 701, Name: "Window director"}
		winner := &film.Film{Id: 4101, Title: "Winner", Year: 2010, Rating: 1400}
		near1 := &film.Film{Id: 4102, Title: "Near low", Year: 2010, Rating: 1400 - testWindow + 50}
		near2 := &film.Film{Id: 4103, Title: "Near high", Year: 2010, Rating: 1400 + testWindow - 50}
		far := &film.Film{Id: 4104, Title: "Far", Year: 2010, Rating: 5000}
		seed(t, txCtx, dir, winner, near1, near2, far)

		got, err := repo.FindCandidates(txCtx, int32(winner.Id), 1, 40, testWindow)
		if err != nil {
			t.Fatalf("FindCandidates: %v", err)
		}
		assertCandidates(t, got, near1.Id, near2.Id)
	})

	t.Run("pads a sparse window up to the minimum", func(t *testing.T) {
		// Regression: two films drawn far ahead of the pack must not be
		// locked into dueling only each other.
		txCtx, _ := newTx(t)
		dir := film.Director{Id: 702, Name: "Leaders director"}
		leader := &film.Film{Id: 4201, Title: "Leader", Year: 2010, Rating: 3000}
		rival := &film.Film{Id: 4202, Title: "Rival", Year: 2010, Rating: 2950}
		pack1 := &film.Film{Id: 4203, Title: "Pack 1", Year: 2010, Rating: 1400}
		pack2 := &film.Film{Id: 4204, Title: "Pack 2", Year: 2010, Rating: 1390}
		pack3 := &film.Film{Id: 4205, Title: "Pack 3", Year: 2010, Rating: 1380}
		seed(t, txCtx, dir, leader, rival, pack1, pack2, pack3)

		got, err := repo.FindCandidates(txCtx, int32(leader.Id), 3, 40, testWindow)
		if err != nil {
			t.Fatalf("FindCandidates: %v", err)
		}
		assertCandidates(t, got, rival.Id, pack1.Id, pack2.Id)
	})

	t.Run("falls back to the closest films when none is in the window", func(t *testing.T) {
		txCtx, _ := newTx(t)
		dir := film.Director{Id: 703, Name: "Outlier director"}
		outlier := &film.Film{Id: 4301, Title: "Outlier", Year: 2010, Rating: 9000}
		closest := &film.Film{Id: 4302, Title: "Closest", Year: 2010, Rating: 1400}
		farther := &film.Film{Id: 4303, Title: "Farther", Year: 2010, Rating: 1300}
		seed(t, txCtx, dir, outlier, closest, farther)

		got, err := repo.FindCandidates(txCtx, int32(outlier.Id), 1, 40, testWindow)
		if err != nil {
			t.Fatalf("FindCandidates: %v", err)
		}
		assertCandidates(t, got, closest.Id)
	})

	t.Run("caps the pool at the maximum", func(t *testing.T) {
		txCtx, _ := newTx(t)
		dir := film.Director{Id: 704, Name: "Cap director"}
		winner := &film.Film{Id: 4401, Title: "Winner", Year: 2010, Rating: 1400}
		seed(t, txCtx, dir, winner)
		inWindow := []*film.Film{
			{Id: 4402, Title: "C1", Year: 2010, Rating: 1410},
			{Id: 4403, Title: "C2", Year: 2010, Rating: 1420},
			{Id: 4404, Title: "C3", Year: 2010, Rating: 1430},
			{Id: 4405, Title: "C4", Year: 2010, Rating: 1440},
			{Id: 4406, Title: "C5", Year: 2010, Rating: 1450},
		}
		seed(t, txCtx, dir, inWindow...)

		got, err := repo.FindCandidates(txCtx, int32(winner.Id), 1, 3, testWindow)
		if err != nil {
			t.Fatalf("FindCandidates: %v", err)
		}
		assertCandidates(t, got, 4402, 4403, 4404)
	})

	t.Run("carries popularity and duel_count into candidates", func(t *testing.T) {
		txCtx, tx := newTx(t)
		dir := film.Director{Id: 705, Name: "Stats director"}
		winner := &film.Film{Id: 4501, Title: "Winner", Year: 2010, Rating: 1400}
		other := &film.Film{Id: 4502, Title: "Other", Year: 2010, Rating: 1410}
		seed(t, txCtx, dir, winner, other)
		if _, err := tx.Exec(ctx,
			"UPDATE films SET popularity = 5.5, duel_count = 7 WHERE id = $1", other.Id,
		); err != nil {
			t.Fatalf("updating film stats: %v", err)
		}

		got, err := repo.FindCandidates(txCtx, int32(winner.Id), 1, 40, testWindow)
		if err != nil {
			t.Fatalf("FindCandidates: %v", err)
		}
		assertCandidates(t, got, other.Id)
		if got[0].Popularity != 5.5 || got[0].DuelCount != 7 {
			t.Errorf("candidate = %+v, want Popularity 5.5 and DuelCount 7", got[0])
		}
		if got[0].CreatedAt.IsZero() {
			t.Errorf("candidate CreatedAt is zero, want the films.created_at value")
		}
	})

	t.Run("unknown winner returns no candidates", func(t *testing.T) {
		txCtx, _ := newTx(t)
		got, err := repo.FindCandidates(txCtx, 999999, 1, 40, testWindow)
		if err != nil {
			t.Fatalf("FindCandidates: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d candidates for unknown winner, want 0", len(got))
		}
	})
}
