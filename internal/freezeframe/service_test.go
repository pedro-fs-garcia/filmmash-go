package freezeframe_test

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/freezeframe"
	"log/slog"
	"testing"
)

func cleanupSeededGame(t *testing.T, ctx context.Context, gameID int32, films []film.Film) {
	t.Helper()
	filmIDs := make([]int32, len(films))
	for i, f := range films {
		filmIDs[i] = int32(f.Id)
	}

	stmts := []struct {
		query string
		arg   any
	}{
		{"DELETE FROM reel_frames WHERE reel_id IN (SELECT id FROM reels WHERE game_id = $1)", gameID},
		{"DELETE FROM reel_alternatives WHERE reel_id IN (SELECT id FROM reels WHERE game_id = $1)", gameID},
		{"DELETE FROM frames WHERE film_id = ANY($1)", filmIDs},
		{"DELETE FROM reels WHERE game_id = $1", gameID},
		{"DELETE FROM games WHERE id = $1", gameID},
		{"DELETE FROM films WHERE id = ANY($1)", filmIDs},
		{"DELETE FROM directors WHERE id = ANY($1)", filmIDs},
	}
	for _, s := range stmts {
		if _, err := testPool.Exec(ctx, s.query, s.arg); err != nil {
			t.Errorf("cleanup: %s: %v", s.query, err)
		}
	}
}

func TestSeedGame(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	g := BuildValidGame()
	gameFilms := gameFilms(g)
	seedFilms(t, ctx, gameFilms)
	t.Cleanup(func() { cleanupSeededGame(t, context.Background(), g.ID, gameFilms) })

	logger := slog.New(&slog.JSONHandler{})

	txm := database.NewTxManager(testPool)
	repo := freezeframe.NewRepository(testPool)
	service := freezeframe.NewService(logger, repo, txm)

	t.Run("SeedGame", func(t *testing.T) {
		err := service.SeedGame(ctx, &g)
		if err != nil {
			t.Fatalf("seeding game: %v", err)
		}
	})
}
