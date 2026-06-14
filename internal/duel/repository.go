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

func NewRepository(pool *pgxpool.Pool) *repository {
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

func (r *repository) GetDuelRatingsForUpdate(ctx context.Context, id uuid.UUID) ([2]film.FilmRating, error) {
	query := `
	SELECT id, rating
	FROM films
	WHERE films.id IN (
		SELECT film_a_id FROM duels WHERE id = $1
		UNION
		SELECT film_b_id FROM duels WHERE id = $1
	)
	ORDER BY films.id
	FOR UPDATE`

	q := database.ExtractTx(ctx, r.pool)
	rows, err := q.Query(ctx, query, id)
	if err != nil {
		log.Println(err)
		return [2]film.FilmRating{}, err
	}
	defer rows.Close()

	var ratings [2]film.FilmRating
	i := 0
	for rows.Next() {
		var film_id int
		var rating float64
		err = rows.Scan(&film_id, &rating)
		if err != nil {
			log.Println(err)
			return [2]film.FilmRating{}, err
		}

		ratings[i] = film.FilmRating{
			Id:     film_id,
			Rating: rating,
		}

		i++
	}

	if err = rows.Err(); err != nil {
		return [2]film.FilmRating{}, err
	}
	if i < 2 {
		return [2]film.FilmRating{}, fmt.Errorf("duel %s does not have two films", id)
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
		WHERE id <> $1
		ORDER BY RANDOM() LIMIT 1
	),
	inserted_duel AS (
		INSERT INTO duels (film_a_id, film_b_id)
		SELECT $1, id FROM random_id
		RETURNING id AS duel_id, film_a_id, film_b_id
	)
	SELECT i.duel_id,
		fa.id, fa.title, fa.release_year, fa.image_path, fa.rating, da.id, da.name,
		fb.id, fb.title, fb.release_year, fb.image_path, fb.rating, db.id, db.name
	FROM inserted_duel i
	JOIN films fa ON fa.id = i.film_a_id JOIN directors da ON da.id = fa.director_id
	JOIN films fb ON fb.id = i.film_b_id JOIN directors db ON db.id = fb.director_id
	`

	var duelId uuid.UUID
	var fa, fb film.Film

	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query, winnerId).Scan(
		&duelId,
		&fa.Id, &fa.Title, &fa.Year, &fa.ImagePath, &fa.Rating, &fa.Director.Id, &fa.Director.Name,
		&fb.Id, &fb.Title, &fb.Year, &fb.ImagePath, &fb.Rating, &fb.Director.Id, &fb.Director.Name,
	)
	if err != nil {
		log.Println(err)
		return Duel{}, err
	}

	return Duel{
		Id:    duelId,
		FilmA: &fa,
		FilmB: &fb,
	}, nil
}

func (r *repository) CountPending(ctx context.Context) (int, error) {
	query := `
	SELECT COUNT(*) FROM duels d
	WHERE NOT EXISTS (SELECT 1 FROM votes v WHERE v.duel_id = d.id)
	`
	var n int
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query).Scan(&n)
	return n, err
}
