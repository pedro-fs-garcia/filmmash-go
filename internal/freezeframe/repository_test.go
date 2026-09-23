package freezeframe_test

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/freezeframe"
	"filmmash/internal/testdb"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

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
