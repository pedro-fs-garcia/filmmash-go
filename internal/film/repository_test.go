package film_test

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/testdb"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

var testDirector = film.Director{
	Id:   543,
	Name: "Testing director",
}

var testFilm = film.Film{
	Id:        1234,
	Title:     "Testing insert",
	Year:      2026,
	Director:  testDirector,
	ImagePath: "test_image_path.jpg",
}

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
	repo := film.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)

	t.Run("InsertDirector", func(t *testing.T) {
		d := testDirector
		if err := repo.InsertDirector(txCtx, &d); err != nil {
			t.Fatalf("inserting director: %v", err)
		}

		got, err := repo.GetDirector(txCtx, d.Id)
		if err != nil {
			t.Fatalf("getting director %v", err)
		}
		want := testDirector
		if got != want {
			t.Errorf("GetDirector() = %+v, want %+v", got, want)
		}
	})

	t.Run("InsertDirector duplicate should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		d2 := testDirector
		err = repo.InsertDirector(nctx, &d2)
		if err == nil {
			t.Fatal("second Insert: got nil, want a unique violation error")
		}

		if !errors.Is(err, film.ErrDuplicateEntry) {
			t.Fatalf("error is not ErrDuplicateEntry: %v", err)
		}
	})

	t.Run("InsertDirector invalid input should fail", func(t *testing.T) {
		cases := map[string]film.Director{
			"name too long":  {Id: 123, Name: strings.Repeat("N", 128)},
			"name too short": {Id: 124, Name: "a"},
			"empty name":     {Id: 125, Name: ""},
		}

		for _, d := range cases {
			itx, err := testPool.Begin(ctx)
			if err != nil {
				t.Fatalf("NewTx: %v", err)
			}
			defer itx.Rollback(ctx)
			txCtx := database.InjectTx(ctx, itx)
			err = repo.InsertDirector(txCtx, &d)
			if err == nil {
				t.Fatalf("expected ErrFilmInvalidInput, got nil")
			}
			if !errors.Is(err, film.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %s", err.Error())
			}
		}
	})

	t.Run("GetDirector unexisting director should fail", func(t *testing.T) {
		_, err := repo.GetDirector(txCtx, 9999)
		if err == nil {
			t.Fatal("Got no error, want ErrNotFound")
		}
		if !errors.Is(err, film.ErrNotFound) {
			t.Fatalf("Got '%v', want 'ErrNotFound'", err)
		}
	})

	t.Run("InsertFilm", func(t *testing.T) {
		f := testFilm
		if err := repo.InsertFilm(txCtx, &f); err != nil {
			t.Fatalf("Insert: %v", err)
		}

		got, err := repo.GetFilm(txCtx, testFilm.Id)
		if err != nil {
			t.Fatalf("GetFilm: %v", err)
		}

		want := testFilm
		want.Rating = film.StdRating
		if got != want {
			t.Errorf("GetFilm() = %+v, want %+v", got, want)
		}
	})

	t.Run("InsertFilm duplicate should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin save point: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		dup := testFilm
		err = repo.InsertFilm(nctx, &dup)
		if err == nil {
			t.Fatal("second Insert: got nil, want a unique violation error")
		}

		if !errors.Is(err, film.ErrDuplicateEntry) {
			t.Fatalf("error is not *pgconn.PgError: %v", err)
		}
	})

	t.Run("InsertFilm director already exists", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin save point: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		f := film.Film{
			Id:       1235,
			Title:    "Testing insert",
			Year:     2026,
			Director: testDirector,
		}
		err = repo.InsertFilm(nctx, &f)
		if err != nil {
			if errors.Is(err, film.ErrDuplicateEntry) {
				t.Fatalf("Trying to reinsert existing director: %v", err)
			}
			t.Fatalf("Insert film: %v", err)
		}
	})

	t.Run("InsertFilm invalid input should fail", func(t *testing.T) {
		cases := map[string]film.Film{
			"title too long": {Id: 123, Title: strings.Repeat("N", 256), Year: 2010, Director: testDirector},
			"year < 1800":    {Id: 124, Title: strings.Repeat("N", 154), Year: 1799, Director: testDirector},
			"year > 2100":    {Id: 125, Title: strings.Repeat("N", 155), Year: 2101, Director: testDirector},
		}

		for name, f := range cases {
			t.Run(name, func(t *testing.T) {
				itx, err := tx.Begin(ctx)
				if err != nil {
					t.Fatalf("NewTx: %v", err)
				}
				defer itx.Rollback(ctx)
				txCtx := database.InjectTx(ctx, itx)
				err = repo.InsertFilm(txCtx, &f)
				if err == nil {
					t.Fatalf("expected ErrInvalidInput, got nil")
				}
				if !errors.Is(err, film.ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %s", err.Error())
				}
			})
		}
	})

	t.Run("GetFilm unexistent film should fail", func(t *testing.T) {
		_, err := repo.GetFilm(txCtx, 9999)
		if err == nil {
			t.Fatal("Got no error, want ErrNotFound")
		}
		if !errors.Is(err, film.ErrNotFound) {
			t.Fatalf("Got '%v', want 'ErrNotFound'", err)
		}
	})

	t.Run("UpdateRating", func(t *testing.T) {
		f := testFilm
		const newRating float64 = 1340.0
		f.Rating = newRating
		if err := repo.UpdateRating(txCtx, &f); err != nil {
			t.Fatalf("UpdateRating: %v", err)
		}

		got, err := repo.GetFilm(txCtx, f.Id)
		if err != nil {
			t.Fatalf("UpdateRating: %v", err)
		}
		if got.Rating != newRating {
			t.Fatalf("UpdateRating got %f, want %f", got.Rating, newRating)
		}
	})

	t.Run("UpdateRating unexisting film should fail", func(t *testing.T) {
		f := testFilm
		f.Id = 1000
		err := repo.UpdateRating(txCtx, &f)
		if err == nil {
			t.Fatalf("Got no error, want ErrNotFound")
		}
		if !errors.Is(err, film.ErrNotFound) {
			t.Fatalf("Got '%v', want 'ErrNotFound'", err)
		}
	})

	queryFilms := []film.Film{
		{Id: 2001, Title: "Alpha", Year: 2000, Director: testDirector, ImagePath: "alpha.jpg", Rating: 1500},
		{Id: 2002, Title: "Beta", Year: 2001, Director: testDirector, ImagePath: "beta.jpg", Rating: 1300},
		{Id: 2003, Title: "Alpha Two", Year: 2002, Director: testDirector, ImagePath: "alpha2.jpg", Rating: 1200},
	}
	for _, f := range queryFilms {
		if err := repo.InsertFilm(txCtx, &f); err != nil {
			t.Fatalf("seeding film %d: %v", f.Id, err)
		}
	}

	t.Run("GetRandomFilm", func(t *testing.T) {
		valid := map[int]bool{1234: true, 2001: true, 2002: true, 2003: true}
		f, err := repo.GetRandomFilm(txCtx)
		if err != nil {
			t.Fatalf("GetRandomFilm: %v", err)
		}
		if !valid[f.Id] {
			t.Fatalf("GetRandomFilm returned unexpected id %d", f.Id)
		}
		if f.Director.Id != testDirector.Id {
			t.Errorf("GetRandomFilm director = %d, want %d", f.Director.Id, testDirector.Id)
		}
	})

	t.Run("GetFilmsPaginatedByRating first page", func(t *testing.T) {
		films, err := repo.GetFilmsPaginatedByRating(txCtx, film.PaginationParameters{Size: 2})
		if err != nil {
			t.Fatalf("GetFilmsPaginatedByRating: %v", err)
		}
		assertFilmIDs(t, films, []int{2001, 1234})
	})

	t.Run("GetFilmsPaginatedByRating with cursor", func(t *testing.T) {
		films, err := repo.GetFilmsPaginatedByRating(txCtx, film.PaginationParameters{
			Size:           2,
			LastSeenId:     1234,
			LastSeenRating: 1340,
		})
		if err != nil {
			t.Fatalf("GetFilmsPaginatedByRating: %v", err)
		}
		assertFilmIDs(t, films, []int{2002, 2003})
	})

	t.Run("SearchFilmByName matches", func(t *testing.T) {
		films, err := repo.SearchFilmByName(txCtx, "Alpha")
		if err != nil {
			t.Fatalf("SearchFilmByName: %v", err)
		}
		got := map[int]bool{}
		for _, f := range films {
			got[f.Id] = true
		}
		if len(films) != 2 || !got[2001] || !got[2003] {
			t.Fatalf("SearchFilmByName(Alpha) = %v, want films 2001 and 2003", filmIDs(films))
		}
	})

	t.Run("SearchFilmByName no match returns empty", func(t *testing.T) {
		films, err := repo.SearchFilmByName(txCtx, "no such title zzz")
		if err != nil {
			t.Fatalf("SearchFilmByName: %v", err)
		}
		if len(films) != 0 {
			t.Fatalf("SearchFilmByName = %v, want empty", filmIDs(films))
		}
	})

	t.Run("UpdateRatings", func(t *testing.T) {
		stx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer stx.Rollback(ctx)
		sctx := database.InjectTx(ctx, stx)

		updates := []*film.FilmRating{
			{Id: 2001, Rating: 999},
			{Id: 2002, Rating: 888},
		}
		if err := repo.UpdateRatings(sctx, updates); err != nil {
			t.Fatalf("UpdateRatings: %v", err)
		}
		for _, u := range updates {
			got, err := repo.GetFilm(sctx, u.Id)
			if err != nil {
				t.Fatalf("GetFilm %d: %v", u.Id, err)
			}
			if got.Rating != u.Rating {
				t.Errorf("film %d rating = %f, want %f", u.Id, got.Rating, u.Rating)
			}
		}
	})

	t.Run("UpdateRatings empty slice is a no-op", func(t *testing.T) {
		if err := repo.UpdateRatings(txCtx, nil); err != nil {
			t.Fatalf("UpdateRatings(nil) = %v, want nil", err)
		}
	})

	t.Run("UpdateRatings unexisting film should fail", func(t *testing.T) {
		stx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer stx.Rollback(ctx)
		sctx := database.InjectTx(ctx, stx)

		err = repo.UpdateRatings(sctx, []*film.FilmRating{{Id: 999999, Rating: 1000}})
		if err == nil {
			t.Fatal("UpdateRatings: got nil, want ErrNotFound")
		}
		if !errors.Is(err, film.ErrNotFound) {
			t.Fatalf("Got '%v', want ErrNotFound", err)
		}
	})

	t.Run("CountTotal", func(t *testing.T) {
		wantCount := int64(1 + len(queryFilms))
		got, err := repo.CountTotal(txCtx)
		if err != nil {
			t.Fatalf("CountTotal: %v", err)
		}
		if got != wantCount {
			t.Fatalf("CountTotal got %d, want %d", got, wantCount)
		}
	})
}

func assertFilmIDs(t *testing.T, films []film.Film, want []int) {
	t.Helper()
	if len(films) != len(want) {
		t.Fatalf("got %v films, want %v", filmIDs(films), want)
	}
	for i, id := range want {
		if films[i].Id != id {
			t.Fatalf("film order = %v, want %v", filmIDs(films), want)
		}
	}
}

func filmIDs(films []film.Film) []int {
	ids := make([]int, len(films))
	for i, f := range films {
		ids[i] = f.Id
	}
	return ids
}
