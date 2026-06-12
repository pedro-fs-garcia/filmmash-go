package duel

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func newRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

func (r *repository) Insert(ctx context.Context, duel *Duel) error {
	query := `
	INSERT INTO duels (film_a_id, film_b_id) 
	VALUES ($1, $2) 
	RETURNING id`

	q := database.ExtractTx(ctx, r.pool)
	return q.QueryRow(ctx, query, duel.FilmA.Id, duel.FilmB.Id).Scan(&duel.Id)
}

func (r *repository) GetById(ctx context.Context, id uuid.UUID) (Duel, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	WHERE films.id IN (
		SELECT film_a_id FROM duels WHERE id = $1
		UNION
		SELECT film_b_id FROM duels WHERE id = $1
	)`

	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, id)
	if err != nil {
		log.Println(err)
		return Duel{}, err
	}
	defer rows.Close()

	var films [2]film.Film
	i := 0
	for rows.Next() {
		var film_id, release_year, director_id int
		var rating float64
		var title, image_path, director_name string
		err = rows.Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)
		if err != nil {
			log.Println(err)
			return Duel{}, err
		}

		films[i] = film.Film{
			Id:    film_id,
			Title: title,
			Year:  release_year,
			Director: film.Director{
				Id:   director_id,
				Name: director_name,
			},
			ImagePath: image_path,
			Rating:    rating,
		}

		i++
	}

	if err = rows.Err(); err != nil {
		return Duel{}, err
	}

	return Duel{
		Id:    id,
		FilmA: &films[0],
		FilmB: &films[1],
	}, nil
}

func (r *repository) GetDuelRatings(ctx context.Context, id uuid.UUID) ([2]FilmRating, error) {
	query := `
	SELECT id, rating
	FROM films
	WHERE films.id IN (
		SELECT film_a_id FROM duels WHERE id = $1
		UNION
		SELECT film_b_id FROM duels WHERE id = $1
	)`

	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, id)
	if err != nil {
		log.Println(err)
		return [2]FilmRating{}, err
	}
	defer rows.Close()

	var ratings [2]FilmRating
	i := 0
	for rows.Next() {
		var film_id int
		var rating float64
		err = rows.Scan(&film_id, &rating)
		if err != nil {
			log.Println(err)
			return [2]FilmRating{}, err
		}

		ratings[i] = FilmRating{
			Id:     film_id,
			Rating: rating,
		}

		i++
	}

	if err = rows.Err(); err != nil {
		return [2]FilmRating{}, err
	}
	if i < 2 {
		return [2]FilmRating{}, fmt.Errorf("duel %s does not have two films", id)
	}
	return ratings, nil
}

func (r *repository) SelectRandomFilms(ctx context.Context) ([2]film.Film, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	ORDER BY RANDOM() LIMIT 2`

	filRows, err := r.pool.Query(ctx, query)
	if err != nil {
		log.Println(err)
		return [2]film.Film{}, err
	}
	defer filRows.Close()

	var films [2]film.Film
	i := 0
	for filRows.Next() {
		var film_id, release_year, director_id int
		var rating float64
		var title, image_path, director_name string
		err = filRows.Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)
		if err != nil {
			log.Println(err)
			return [2]film.Film{}, err
		}

		films[i] = film.Film{
			Id:    film_id,
			Title: title,
			Year:  release_year,
			Director: film.Director{
				Id:   director_id,
				Name: director_name,
			},
			ImagePath: image_path,
			Rating:    rating,
		}

		i++
	}
	if err = filRows.Err(); err != nil {
		return [2]film.Film{}, err
	}
	if i < 2 {
		return [2]film.Film{}, fmt.Errorf("not enough films for a duel: got %d", i)
	}
	return films, nil
}
