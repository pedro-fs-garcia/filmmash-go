package film

import (
	"context"
	"filmmash/internal/database"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (r *repository) InsertFilm(ctx context.Context, f *Film) error {
	query := `
	WITH new_director AS (
		INSERT INTO directors (id, name)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		RETURNING id AS director_id
	),
	director AS (
		SELECT director_id FROM new_director
		UNION ALL
		SELECT id AS director_id FROM directors WHERE id = $1
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
	return parseDBError("inserting film", err)
}

func (r *repository) InsertDirector(ctx context.Context, d *Director) error {
	query := `
	INSERT INTO directors (id, name)
	VALUES ($1, $2)
	RETURNING id
	`
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query, d.Id, d.Name).Scan(&d.Id)

	return parseDBError(
		fmt.Sprintf("inserting director (id: %v)", d.Id),
		err,
	)
}

func (r *repository) GetDirector(ctx context.Context, id int) (Director, error) {
	query := "SELECT id, name FROM directors WHERE id = $1"
	q := database.ExtractTx(ctx, r.pool)
	d := Director{}
	err := q.QueryRow(ctx, query, id).Scan(&d.Id, &d.Name)
	if err != nil {
		return Director{}, parseDBError(
			fmt.Sprintf("getting director by id (id: %d)", id),
			err,
		)
	}
	return d, nil
}

func (r *repository) GetFilm(ctx context.Context, id int) (Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	WHERE films.id = $1`

	q := database.ExtractTx(ctx, r.pool)

	var f Film
	err := q.QueryRow(ctx, query, id).Scan(
		&f.Id, &f.Title, &f.Year, &f.ImagePath,
		&f.Rating, &f.Director.Id, &f.Director.Name,
	)
	if err != nil {
		return Film{}, parseDBError(
			fmt.Sprintf("getting film by id (id: %d)", id),
			err,
		)
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
		return Film{}, parseDBError(
			"getting random film",
			err,
		)
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
		return nil, parseDBError("querying paginated films", err)
	}
	defer rows.Close()

	var films []Film
	for rows.Next() {
		var f Film
		err = rows.Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
		if err != nil {
			return nil, parseDBError("scanning film row", err)
		}
		films = append(films, f)
	}
	if err = rows.Err(); err != nil {
		return nil, parseDBError("iterating rows", err)
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
		return nil, parseDBError("querying films by name", err)
	}
	defer rows.Close()

	var films []Film
	for rows.Next() {
		var f Film
		err = rows.Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
		if err != nil {
			return nil, parseDBError("scanning row output to film", err)
		}
		films = append(films, f)
	}
	if err = rows.Err(); err != nil {
		return nil, parseDBError("iterating rows", err)
	}
	return films, nil
}

func (r *repository) UpdateRating(ctx context.Context, f *Film) error {
	query := "UPDATE films SET rating = $1 WHERE id = $2"

	q := database.ExtractTx(ctx, r.pool)
	tag, err := q.Exec(ctx, query, f.Rating, f.Id)
	if err != nil {
		return parseDBError(fmt.Sprintf("updating rating for film(id: %d)", f.Id), err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("film not found to update rating (id: %d): %w", f.Id, ErrNotFound)
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
	for range films {
		tag, err := res.Exec()
		if err != nil {
			return parseDBError("updating ratings for film batch", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("no register affected on batch update: %w", ErrNotFound)
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
		return 0, parseDBError("counting films", err)
	}
	return n, nil
}
