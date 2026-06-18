package film

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
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
				"[film.repository.Insert] conflict inserting film or director (film_id: %v, director_id: %v): %w",
				f.Id, f.Director.Id, ErrDuplicateEntry,
			)
		}
		return fmt.Errorf("[film.repository.Insert] Failed to insert film (id: %d): %w", f.Id, err)
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

	var f Film
	err := q.QueryRow(ctx, query, id).Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Film{}, fmt.Errorf("[film.repository.GetFilm] Repository failed to get film (id: %d): %w", id, ErrNotFound)
		}
		return Film{}, fmt.Errorf("[film.repository.GetFilm] Internal database error to get film (id: %d): %w", id, err)
	}
	return f, nil
}

func (r *repository) GetRandomFilm(ctx context.Context) (Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	ORDER BY RANDOM() LIMIT 1`

	var f Film
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query).Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Film{}, fmt.Errorf("[film.repository.GetRandomFilm] Empty database; no film available: %w", ErrNotFound)
		}
		return Film{}, fmt.Errorf("[film.repository.GetRandomFilm] Failed to get random film: %w", err)
	}

	return f, nil
}

func (r *repository) GetFilmsPaginatedByRating(ctx context.Context, pars PaginationParameters) ([]Film, error) {
	var (
		where string
		args  []any
	)

	if pars.LastSeenId != 0 {
		where = "WHERE (films.rating, films.id) < ($2, $3)"
		args = []any{pars.Size, pars.LastSeenRating, pars.LastSeenId}
	} else {
		args = []any{pars.Size}
	}

	query := fmt.Sprintf(`
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	%s
	ORDER BY films.rating DESC, films.id DESC
	LIMIT $1
	`, where)

	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return []Film{}, fmt.Errorf("[film.repository.GetFilmsPaginated] Error querying paginated films: %w", err)
	}
	defer rows.Close()

	var films []Film
	for rows.Next() {
		var f Film
		err = rows.Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
		if err != nil {
			return []Film{}, fmt.Errorf("[film.repository.GetFilmsPaginated] Internal database error to get paginated films: %w", err)
		}
		films = append(films, f)
	}
	if err = rows.Err(); err != nil {
		return []Film{}, fmt.Errorf("[film.repository.GetFilmsPaginated] iterating rows: %w", err)
	}
	return films, nil
}

func (r *repository) SearchFilmByName(ctx context.Context, search string) ([]Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	WHERE films.title ILIKE '%' || $1 || '%'
	`
	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, search)
	if err != nil {
		return []Film{}, fmt.Errorf("[film.repository.SearchFilmByName] error querying search film by name: %w", err)
	}
	defer rows.Close()

	var films []Film
	for rows.Next() {
		var f Film
		err = rows.Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
		if err != nil {
			return []Film{}, fmt.Errorf("[film.repository.SearchFilmByName] internal database error to search film by name: %w", err)
		}
		films = append(films, f)
	}
	if err = rows.Err(); err != nil {
		return []Film{}, fmt.Errorf("[film.repository.SearchFilmByName] iterating rows: %w", err)
	}
	return films, nil
}

func (r *repository) UpdateRating(ctx context.Context, f *Film) error {
	query := "UPDATE films SET rating = $1 WHERE id = $2"

	q := database.ExtractTx(ctx, r.pool)
	tag, err := q.Exec(ctx, query, f.Rating, f.Id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return fmt.Errorf("[film.repository.UpdateRating] Could not update film rating due to invalid rating value (rating: %f): %w", f.Rating, ErrInvalidRating)
		}
		return fmt.Errorf("[film.repository.UpdateRating] Failed to upload film rating: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("[film.repository.UpdateRating] No film was found to update rating (id: %d): %w", f.Id, ErrNotFound)
	}
	return nil
}

func (r *repository) UpdateRatings(ctx context.Context, films []*FilmRating) error {
	if len(films) == 0 {
		return nil
	}

	query := "UPDATE films SET rating = $1 WHERE id = $2"

	batch := &pgx.Batch{}
	for i := range films {
		batch.Queue(query, films[i].Rating, films[i].Id)
	}
	q := database.ExtractTx(ctx, r.pool)
	res := q.SendBatch(ctx, batch)
	defer res.Close()
	for _, f := range films {
		tag, err := res.Exec()

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23514" {
				return fmt.Errorf("[film.repository.UpdateRatings] Could not update film rating due to invalid rating value (rating: %f): %w", f.Rating, ErrInvalidRating)
			}
			return fmt.Errorf("[film.repository.UpdateRatings] Failed to upload film ratings batch: %w", err)
		}

		if tag.RowsAffected() == 0 {
			return fmt.Errorf("[film.repository.UpdateRatings] No register affected on batch update: %w", ErrNotFound)
		}
	}
	return nil
}

func (r *repository) CountTotal(ctx context.Context) (int64, error) {
	query := "SELECT COUNT(*) FROM films"
	q := database.ExtractTx(ctx, r.pool)
	var n int64
	err := q.QueryRow(ctx, query).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("[film.repository.CountTotal] Failed to count films: %w", err)
	}
	return n, nil
}
