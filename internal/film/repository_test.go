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

func TestRepository_InsertDirector(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	repo := film.NewRepository(testPool)

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
}

func TestRepository_InsertDirector_DuplicateFail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("NewTx: %v", err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	repo := film.NewRepository(testPool)

	d1 := testDirector
	if err := repo.InsertDirector(txCtx, &d1); err != nil {
		t.Fatalf("first Insert: %v", err)
	}

	d2 := testDirector
	err = repo.InsertDirector(txCtx, &d2)
	if err == nil {
		t.Fatal("second Insert: got nil, want a unique violation error")
	}

	if !errors.Is(err, film.ErrDuplicateEntry) {
		t.Fatalf("error is not ErrDuplicateEntry: %v", err)
	}
}

func TestRepository_InsertDirector_InvalidNameFail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("NewTx: %v", err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	repo := film.NewRepository(testPool)
	d1 := film.Director{Id: 234, Name: strings.Repeat("N", 128)}
	err = repo.InsertDirector(txCtx, &d1)
	if err == nil {
		t.Fatalf("got nil, expected a restraint violation err")
	}
	if !errors.Is(err, film.ErrInvalidInput) {
		t.Fatalf("err is not ErrInvalidInput")
	}

	d1.Name = ""
	err = repo.InsertDirector(txCtx, &d1)
	if err == nil {
		t.Fatalf("got nil, expected a restraint violation err")
	}
	if !errors.Is(err, film.ErrInvalidInput) {
		t.Fatalf("err is not ErrInvalidInput")
	}
}

func TestRepository_InsertFilm(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("NewTx: %v", err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	repo := film.NewRepository(testPool)

	f := testFilm
	if err := repo.InsertFilm(txCtx, &f); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := repo.GetFilm(txCtx, testFilm.Id)
	if err != nil {
		t.Fatalf("GetFilm: %v", err)
	}

	want := testFilm
	want.Rating = 1400 // schema default for films.rating
	if got != want {
		t.Errorf("GetFilm() = %+v, want %+v", got, want)
	}
}

func TestRepository_Insert_DuplicateFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("NewTx: %v", err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	repo := film.NewRepository(testPool)

	f := testFilm
	if err := repo.InsertFilm(txCtx, &f); err != nil {
		t.Fatalf("first Insert: %v", err)
	}

	dup := testFilm
	err = repo.InsertFilm(txCtx, &dup)
	if err == nil {
		t.Fatal("second Insert: got nil, want a unique violation error")
	}

	if !errors.Is(err, film.ErrDuplicateEntry) {
		t.Fatalf("error is not *pgconn.PgError: %v", err)
	}
}

func Test_InsertFilm_Director_Already_Exists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("NewTx: %v", err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)
	repo := film.NewRepository(testPool)

	d := testDirector
	f := testFilm
	if err := repo.InsertDirector(txCtx, &d); err != nil {
		t.Fatalf("Inserting director: %v", err)
	}

	err = repo.InsertFilm(txCtx, &f)
	if err != nil {
		if errors.Is(err, film.ErrDuplicateEntry) {
			t.Fatalf("Trying to reinsert existing director: %v", err)
		}
		t.Fatalf("Insert film: %v", err)
	}
}
