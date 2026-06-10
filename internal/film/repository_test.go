package film_test

import (
	"context"
	"filmmash/internal/film"
	"filmmash/internal/testdb"
	"log"
	"os"
	"testing"

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
	ctx := context.Background()
	t.Cleanup(func() {
		testdb.Truncate(ctx, t, testPool, "films", "directors")
	})

	repo := film.NewRepository(testPool)

	f := testFilm
	if err := repo.Insert(ctx, &f); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := repo.GetFilm(ctx, testFilm.Id)
	if err != nil {
		t.Fatalf("GetFilm: %v", err)
	}

	want := testFilm
	want.Rating = 1400 // schema default for films.rating
	if got != want {
		t.Errorf("GetFilm() = %+v, want %+v", got, want)
	}
}
