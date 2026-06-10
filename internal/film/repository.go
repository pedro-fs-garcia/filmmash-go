package film

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{db: pool}
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
	return r.db.QueryRow(ctx,
		query,
		f.Director.Id,
		f.Director.Name,
		f.Id,
		f.Title,
		f.Year,
		f.ImagePath,
	).Scan(&f.Id)
}

func (r *repository) GetFilm(ctx context.Context, id int) (Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	WHERE films.id = $1`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return Film{}, err
	}

	defer rows.Close()

	var film_id, release_year, director_id int
	var rating float64
	var title, image_path, director_name string
	rows.Next()
	err = rows.Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)

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

	var film_id, release_year, director_id int
	var rating float64
	var title, image_path, director_name string
	err := r.db.QueryRow(ctx, query).Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)
	if err != nil {
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
