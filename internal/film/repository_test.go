package film_test

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/testdb"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

var testFilm = film.Film{
	Id:    1234,
	Title: "Testing insert",
	Year:  2026,
	Director: film.Director{
		Id:   543,
		Name: "Testing director",
	},
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

func TestRepository_Insert(t *testing.T) {
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
	if err := repo.Insert(txCtx, &f); err != nil {
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
	if err := repo.Insert(txCtx, &f); err != nil {
		t.Fatalf("first Insert: %v", err)
	}

	dup := testFilm
	err = repo.Insert(txCtx, &dup)
	if err == nil {
		t.Fatal("second Insert: got nil, want a unique violation error")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error is not *pgconn.PgError: %v", err)
	}
	if pgErr.Code != "23505" { // unique_violation
		t.Errorf("SQLSTATE = %s, want 23505 (unique_violation)", pgErr.Code)
	}
}
