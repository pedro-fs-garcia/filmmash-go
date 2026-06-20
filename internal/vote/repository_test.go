package vote_test

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/testdb"
	"filmmash/internal/vote"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedFilms(t *testing.T, ctx context.Context) (film.Film, film.Film, duel.Duel) {
	t.Helper()
	filmRepo := film.NewRepository(testPool)
	duelRepo := duel.NewRepository(testPool)
	var testDirector = film.Director{Id: 543, Name: "Testing director"}
	a := film.Film{Id: 2001, Title: "Alpha", Year: 2000, Director: testDirector, ImagePath: "a.jpg"}
	b := film.Film{Id: 2002, Title: "Beta", Year: 2001, Director: testDirector, ImagePath: "b.jpg"}
	for _, f := range []*film.Film{&a, &b} {
		if err := filmRepo.InsertFilm(ctx, f); err != nil {
			t.Fatalf("seeding film %d: %v", f.Id, err)
		}
	}
	d := duel.Duel{FilmA: &a, FilmB: &b}
	if err := duelRepo.Insert(ctx, &d); err != nil {
		t.Fatalf("seeding duel: %v", err)
	}
	return a, b, d
}

func IsValidUUID(t *testing.T, u string) bool {
	t.Helper()
	parsed, err := uuid.Parse(u)
	if err != nil {
		return false
	}
	if parsed.Version() != 7 || parsed.Variant() != uuid.RFC4122 {
		return false
	}
	return true
}

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

func TestRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := vote.NewRepository(testPool)
	duelRepo := duel.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	fa, fb, d := seedFilms(t, txCtx)

	testVote := vote.Vote{
		DuelId:            d.Id,
		WinnerID:          fa.Id,
		LoserId:           fb.Id,
		WinnerRatingAfter: 1410.0,
		LoserRatingAfter:  1390.0,
	}

	t.Run("InsertVote", func(t *testing.T) {
		v := testVote
		if err := repo.InsertVote(txCtx, &v); err != nil {
			t.Fatalf("InsertVote: %v", err)
		}
		if !IsValidUUID(t, v.Id.String()) {
			t.Fatalf("got %v, want VERSION_7", v.Id.Version())
		}
	})

	t.Run("InsertVote duplicate duel should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		nd := duel.Duel{FilmA: &fa, FilmB: &fb}
		if err := duelRepo.Insert(nctx, &nd); err != nil {
			t.Fatalf("seeding duel: %v", err)
		}

		first := testVote
		first.DuelId = nd.Id
		if err := repo.InsertVote(nctx, &first); err != nil {
			t.Fatalf("first InsertVote: %v", err)
		}

		dup := testVote
		dup.DuelId = nd.Id
		err = repo.InsertVote(nctx, &dup)
		if err == nil {
			t.Fatal("got no error, want ErrDuplicateEntry")
		}
		if !errors.Is(err, vote.ErrDuplicateEntry) {
			t.Fatalf("got %v, want ErrDuplicateEntry", err)
		}
	})

	t.Run("CurrentTotal", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		before, err := repo.CurrentTotal(nctx)
		if err != nil {
			t.Fatalf("CurrentTotal: %v", err)
		}

		nd := duel.Duel{FilmA: &fa, FilmB: &fb}
		if err := duelRepo.Insert(nctx, &nd); err != nil {
			t.Fatalf("seeding duel: %v", err)
		}
		v := testVote
		v.DuelId = nd.Id
		if err := repo.InsertVote(nctx, &v); err != nil {
			t.Fatalf("InsertVote: %v", err)
		}

		after, err := repo.CurrentTotal(nctx)
		if err != nil {
			t.Fatalf("CurrentTotal: %v", err)
		}
		if after != before+1 {
			t.Fatalf("CurrentTotal after insert = %d, want %d", after, before+1)
		}
	})
}
