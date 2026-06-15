package film

import (
	"context"
	"database/sql"
	"errors"
	"filmmash/internal/database"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func NewRepository(logger *slog.Logger, pool *pgxpool.Pool) *repository {
	return &repository{
		logger: logger.With(slog.String("component", "repository")),
		pool:   pool,
	}
}

func (r *repository) Insert(ctx context.Context, f *Film) error {
	query := `
	WITH director AS (
		INSERT INTO directors (id, name)
		VALUES ($1, $2)
		RETURNING id AS director_id
	)
	INSERT INTO films (id, title, release_year, director_id, image_path)
	SELECT $3, $4, $5, director_id, $6 FROM director
	RETURNING id;
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx,
		query,
		f.Director.Id,
		f.Director.Name,
		f.Id,
		f.Title,
		f.Year,
		f.ImagePath,
	).Scan(&f.Id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf(
				"conflito de insercao de filme ou diretor (film_id: %v, director_id: %v): %w",
				f.Id, f.Director.Id, ErrDuplicateEntry,
			)
		}
		return fmt.Errorf("Failed to insert film (id: %d): %w", f.Id, err)
	}
	return nil
}

func (r *repository) GetFilm(ctx context.Context, id int) (Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	WHERE films.id = $1`

	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return Film{}, fmt.Errorf("Repository failed to get film (id: %d): %w", id, ErrNotFound)
		}
		return Film{}, fmt.Errorf("Internal database error to get film (id: %d): %w", id, err)
	}

	defer rows.Close()

	var film_id, release_year, director_id int
	var rating float64
	var title, image_path, director_name string
	rows.Next()
	err = rows.Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)
	if err != nil {
		return Film{}, fmt.Errorf("Failed to map film data (id: %d): %w", id, err)

	}
	film := Film{
		Id:    film_id,
		Title: title,
		Year:  release_year,
		Director: Director{
			Id:   director_id,
			Name: director_name,
		},
		ImagePath: image_path,
		Rating:    rating,
	}
	return film, nil
}

func (r *repository) GetRandomFilm(ctx context.Context) (Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	ORDER BY RANDOM() LIMIT 1`

	q := database.ExtractTx(ctx, r.pool)
	var film_id, release_year, director_id int
	var rating float64
	var title, image_path, director_name string
	err := q.QueryRow(ctx, query).Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to get random film",
			slog.String("error", err.Error()),
		)
		return Film{}, err
	}

	film := Film{
		Id:    film_id,
		Title: title,
		Year:  release_year,
		Director: Director{
			Id:   director_id,
			Name: director_name,
		},
		ImagePath: image_path,
		Rating:    rating,
	}
	return film, nil
}

func (r *repository) UpdateRating(ctx context.Context, f *Film) error {
	query := "UPDATE films SET rating = $1 WHERE id = $2"

	q := database.ExtractTx(ctx, r.pool)
	_, err := q.Exec(ctx, query, f.Rating, f.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) UpdateRatings(ctx context.Context, films []*FilmRating) error {
	query := "UPDATE films SET rating = $1 WHERE id = $2"

	batch := &pgx.Batch{}
	for i := range films {
		batch.Queue(query, films[i].Rating, films[i].Id)
	}
	q := database.ExtractTx(ctx, r.pool)
	res := q.SendBatch(ctx, batch)
	defer res.Close()
	for range films {
		if _, err := res.Exec(); err != nil {
			r.logger.ErrorContext(ctx, "failed to update film ratings batch",
				slog.String("error", err.Error()),
			)
			return err
		}
	}
	return nil
}

func (r *repository) CountTotal(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM films"
	q := database.ExtractTx(ctx, r.pool)
	var n int
	err := q.QueryRow(ctx, query).Scan(&n)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to count films",
			slog.String("error", err.Error()),
		)
		return 0, err
	}
	return n, nil
}
