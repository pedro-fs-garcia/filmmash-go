package freezeframe_test

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/freezeframe"
	"filmmash/internal/testdb"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

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

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testdb.New(ctx)
	if err != nil {
		log.Fatalf("test db setup; %v", err)
	}
	testPool = pool
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestInsertGame(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := freezeframe.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := database.InjectTx(ctx, tx)

	g := BuildValidGame()
	seedFilms(t, txCtx, gameFilms(g))

	t.Run("InsertGame", func(t *testing.T) {
		gameID, err := repo.InsertGame(txCtx, g)
		if err != nil {
			t.Fatalf("inserting game: %v", err)
		}
		if gameID == 0 {
			t.Fatalf("inserted game with id 0")
		}
		g.ID = gameID
	})

	t.Run("InsertReels", func(t *testing.T) {
		reelsIDs, err := repo.InsertReels(txCtx, g.ID, g.Reels)
		if err != nil {
			t.Fatalf("inserting reels: %v", err)
		}
		if len(reelsIDs) != len(g.Reels) {
			t.Fatalf("got %d IDs, wanted %d", len(g.Reels), len(reelsIDs))
		}
		validateIDs(t, reelsIDs, len(g.Reels), "Reels")
		for i, id := range reelsIDs {
			g.Reels[i].ID = id
		}
	})

	t.Run("InsertReelAlternatives", func(t *testing.T) {
		for _, reel := range g.Reels {
			altIDs, err := repo.InsertReelAlternatives(txCtx, reel.ID, reel.Alternatives)
			if err != nil {
				t.Fatalf("inserting alternatives for Reel[%d]", reel.ID)
			}
			validateIDs(t, altIDs, len(reel.Alternatives), "Alternatives")
			for j, id := range altIDs {
				reel.Alternatives[j].ID = id
			}
		}
	})

	t.Run("InsertFrames", func(t *testing.T) {
		for _, reel := range g.Reels {
			frames := make([]freezeframe.Frame, len(reel.ReelFrames))
			for i, rf := range reel.ReelFrames {
				frames[i] = rf.Frame
			}

			frameIDs, err := repo.InsertFrames(txCtx, frames)
			if err != nil {
				t.Fatalf("inserting frames for Reel[%d]: %v", reel.ID, err)
			}
			validateIDs(t, frameIDs, len(frames), "Frames")
			for i, id := range frameIDs {
				reel.ReelFrames[i].Frame.ID = id
			}
		}
	})

	t.Run("InsertReelFrames", func(t *testing.T) {
		for _, reel := range g.Reels {
			reelFrameIDs, err := repo.InsertReelFrames(txCtx, reel.ID, reel.ReelFrames)
			if err != nil {
				t.Fatalf("inserting reel frames for Reel[%d]: %v", reel.ID, err)
			}
			validateIDs(t, reelFrameIDs, len(reel.ReelFrames), "ReelFrames")
			for i, id := range reelFrameIDs {
				reel.ReelFrames[i].ID = id
			}
		}
	})
}
