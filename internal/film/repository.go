package film

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

func (r *repository) GetFilm(ctx context.Context, id int) (Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	WHERE films.id = $1`

	rows, err := r.pool.Query(ctx, query, id)
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
	err := r.pool.QueryRow(ctx, query).Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)
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
