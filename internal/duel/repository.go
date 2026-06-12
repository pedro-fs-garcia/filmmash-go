package duel

import (
	"context"
	"encoding/json"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FilmJSON struct {
	Id           int     `json:"id"`
	Title        string  `json:"title"`
	Year         int     `json:"release_year"`
	ImagePath    string  `json:"image_path"`
	Rating       float64 `json:"rating"`
	DirectorId   int     `json:"director_id"`
	DirectorName string  `json:"director_name"`
}

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

	q := database.ExtractTx(ctx, r.pool)
	filRows, err := q.Query(ctx, query)
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

func (r *repository) ComposeDuel(ctx context.Context, winnerId int) (Duel, error) {
	query := `
	WITH random_id AS (
		SELECT id FROM films
		ORDER BY RANDOM() LIMIT 1
	),
	inserted_duel AS (
		INSERT INTO duels (film_a_id, film_b_id)
		SELECT $1, id FROM random_id
		RETURNING id AS duel_id, film_a_id, film_b_id
	),
	target_films AS (
		SELECT f.id, f.title, f.release_year, f.image_path, f.rating, d.id AS director_id, d.name AS director_name
		FROM films f
		JOIN directors d ON f.director_id = d.id
		WHERE f.id = $1 OR f.id = (SELECT film_b_id FROM inserted_duel)
	)
	SELECT i.duel_id,
		(SELECT row_to_json(t.*) FROM target_films t WHERE t.id = i.film_a_id) AS film_a_data,
		(SELECT row_to_json(t.*) FROM target_films t WHERE t.id = i.film_b_id) AS film_b_data
	FROM inserted_duel i
	`

	var duelId uuid.UUID
	var filmAJSON, filmBJSON []byte

	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query, winnerId).Scan(&duelId, &filmAJSON, &filmBJSON)
	if err != nil {
		log.Println(err)
		return Duel{}, err
	}

	var fa, fb FilmJSON
	if err = json.Unmarshal(filmAJSON, &fa); err != nil {
		log.Println(err)
		return Duel{}, err
	}
	if err = json.Unmarshal(filmBJSON, &fb); err != nil {
		log.Println(err)
		return Duel{}, err
	}

	return Duel{
		Id: duelId,
		FilmA: &film.Film{
			Id:        fa.Id,
			Title:     fa.Title,
			Year:      fa.Year,
			ImagePath: fa.ImagePath,
			Rating:    fa.Rating,
			Director: film.Director{
				Id:   fa.DirectorId,
				Name: fa.DirectorName,
			},
		},
		FilmB: &film.Film{
			Id:        fb.Id,
			Title:     fb.Title,
			Year:      fb.Year,
			ImagePath: fb.ImagePath,
			Rating:    fb.Rating,
			Director: film.Director{
				Id:   fb.DirectorId,
				Name: fb.DirectorName,
			},
		},
	}, nil
}
