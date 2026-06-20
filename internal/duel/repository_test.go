package duel_test

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/testdb"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedFilms(t *testing.T, ctx context.Context) (film.Film, film.Film) {
	t.Helper()
	filmRepo := film.NewRepository(testPool)
	var testDirector = film.Director{Id: 543, Name: "Testing director"}
	a := film.Film{Id: 2001, Title: "Alpha", Year: 2000, Director: testDirector, ImagePath: "a.jpg"}
	b := film.Film{Id: 2002, Title: "Beta", Year: 2001, Director: testDirector, ImagePath: "b.jpg"}
	for _, f := range []*film.Film{&a, &b} {
		if err := filmRepo.InsertFilm(ctx, f); err != nil {
			t.Fatalf("seeding film %d: %v", f.Id, err)
		}
	}
	return a, b
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
		log.Fatalf("test db setup: %v", err)
	}
	testPool = pool
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := duel.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	a, b := seedFilms(t, txCtx)

	testDuel := duel.Duel{FilmA: &a, FilmB: &b}

	t.Run("Insert", func(t *testing.T) {
		d := testDuel
		err := repo.Insert(txCtx, &d)
		if err != nil {
			t.Fatalf("inserting duel: %v", err)
		}

		if !IsValidUUID(t, d.Id.String()) {
			t.Fatalf("got %v, want VERSION_7", d.Id.Version())
		}

		got, err := repo.GetById(txCtx, d.Id)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		want := d
		if !reflect.DeepEqual(got, want) {
			t.Errorf("insert duel got %+v (films %+v / %+v), want %+v (films %+v / %+v)",
				got, got.FilmA, got.FilmB, want, want.FilmA, want.FilmB)
		}
	})

	t.Run("Insert duel with equal films should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		d := duel.Duel{
			FilmA: testDuel.FilmA,
			FilmB: testDuel.FilmA,
		}

		err = repo.Insert(nctx, &d)
		if err == nil {
			t.Fatalf("got no errors, wanted ErrInvalidDuel")
		}
		if !errors.Is(err, duel.ErrInvalidDuel) {
			t.Fatalf("got %v, want ErrInvalidDuel", err)
		}
	})

	t.Run("Insert with invalid film should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: '%v'", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		d := testDuel
		d.FilmA = &film.Film{}
		err = repo.Insert(nctx, &d)
		if err == nil {
			t.Fatalf("got no errors, want ErrInvalidDuel")
		}
		if !errors.Is(err, duel.ErrInvalidDuel) {
			t.Fatalf("got '%v', want ErrInvalidDuel", err)
		}
	})

	t.Run("GetById unknown id should fail", func(t *testing.T) {
		_, err := repo.GetById(txCtx, uuid.New())
		if err == nil {
			t.Fatal("got no error, want ErrNotEnoughFilms")
		}
		if !errors.Is(err, duel.ErrNotEnoughFilms) {
			t.Fatalf("got %v, want ErrNotEnoughFilms", err)
		}
	})

	t.Run("GetDuelRatingsForUpdate", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		d := duel.Duel{FilmA: &a, FilmB: &b}
		if err := repo.Insert(nctx, &d); err != nil {
			t.Fatalf("inserting duel: %v", err)
		}

		ratings, err := repo.GetDuelRatingsForUpdate(nctx, d.Id)
		if err != nil {
			t.Fatalf("GetDuelRatingsForUpdate: %v", err)
		}

		want := [2]film.FilmRating{
			{Id: a.Id, Rating: a.Rating},
			{Id: b.Id, Rating: b.Rating},
		}
		if ratings != want {
			t.Errorf("GetDuelRatingsForUpdate = %+v, want %+v", ratings, want)
		}
	})

	t.Run("GetDuelRatingsForUpdate unknown id should fail", func(t *testing.T) {
		_, err := repo.GetDuelRatingsForUpdate(txCtx, uuid.New())
		if err == nil {
			t.Fatal("got no error, want ErrNotEnoughFilms")
		}
		if !errors.Is(err, duel.ErrNotEnoughFilms) {
			t.Fatalf("got %v, want ErrNotEnoughFilms", err)
		}
	})

	t.Run("SelectRandomFilms", func(t *testing.T) {
		films, err := repo.SelectRandomFilms(txCtx)
		if err != nil {
			t.Fatalf("SelectRandomFilms: %v", err)
		}
		valid := map[int]bool{a.Id: true, b.Id: true}
		if !valid[films[0].Id] || !valid[films[1].Id] {
			t.Errorf("SelectRandomFilms = %d/%d, want a subset of %d/%d",
				films[0].Id, films[1].Id, a.Id, b.Id)
		}
		if films[0].Id == films[1].Id {
			t.Errorf("SelectRandomFilms returned the same film twice: %d", films[0].Id)
		}
	})

	t.Run("ComposeDuel", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		d, err := repo.ComposeDuel(nctx, a.Id)
		if err != nil {
			t.Fatalf("ComposeDuel: %v", err)
		}
		if !IsValidUUID(t, d.Id.String()) {
			t.Fatalf("got %v, want VERSION_7", d.Id.Version())
		}
		if d.FilmA.Id != a.Id {
			t.Errorf("ComposeDuel FilmA = %d, want %d", d.FilmA.Id, a.Id)
		}
		if d.FilmB.Id != b.Id {
			t.Errorf("ComposeDuel FilmB = %d, want %d", d.FilmB.Id, b.Id)
		}
	})

	t.Run("ComposeDuel with no opponent should fail", func(t *testing.T) {
		itx, err := testPool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer itx.Rollback(ctx)
		ictx := database.InjectTx(ctx, itx)

		lone := film.Film{
			Id:        3001,
			Title:     "Lonely",
			Year:      2010,
			Director:  film.Director{Id: 600, Name: "Solo director"},
			ImagePath: "lone.jpg",
		}
		if err := film.NewRepository(testPool).InsertFilm(ictx, &lone); err != nil {
			t.Fatalf("seeding lone film: %v", err)
		}

		_, err = repo.ComposeDuel(ictx, lone.Id)
		if err == nil {
			t.Fatal("got no error, want ErrNotEnoughFilms")
		}
		if !errors.Is(err, duel.ErrNotEnoughFilms) {
			t.Fatalf("got %v, want ErrNotEnoughFilms", err)
		}
	})

	t.Run("CountPending", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		before, err := repo.CountPending(nctx)
		if err != nil {
			t.Fatalf("CountPending: %v", err)
		}

		d1 := duel.Duel{FilmA: &a, FilmB: &b}
		d2 := duel.Duel{FilmA: &a, FilmB: &b}
		for _, d := range []*duel.Duel{&d1, &d2} {
			if err := repo.Insert(nctx, d); err != nil {
				t.Fatalf("inserting duel: %v", err)
			}
		}

		after, err := repo.CountPending(nctx)
		if err != nil {
			t.Fatalf("CountPending: %v", err)
		}
		if after != before+2 {
			t.Fatalf("CountPending after inserts = %d, want %d", after, before+2)
		}

		seedVote(t, ntx, d1.Id, a.Id, b.Id)
		voted, err := repo.CountPending(nctx)
		if err != nil {
			t.Fatalf("CountPending: %v", err)
		}
		if voted != before+1 {
			t.Fatalf("CountPending after vote = %d, want %d", voted, before+1)
		}
	})
}

func seedVote(t *testing.T, q pgx.Tx, duelId uuid.UUID, winnerId, loserId int) {
	t.Helper()
	_, err := q.Exec(context.Background(),
		`INSERT INTO votes (duel_id, winner_id, loser_id, winner_rating_after, loser_rating_after)
		 VALUES ($1, $2, $3, $4, $5)`,
		duelId, winnerId, loserId, 1410.0, 1390.0,
	)
	if err != nil {
		t.Fatalf("seeding vote: %v", err)
	}
}
