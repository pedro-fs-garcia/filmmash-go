package duel

import (
	"context"
	"filmmash/internal/film"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Duel struct {
	Id    uuid.UUID
	FilmA *film.Film
	FilmB *film.Film
}

type Service struct {
	pool        *pgxpool.Pool
	repo        *repository
	filmService *film.Service
}

func NewService(pool *pgxpool.Pool, filmService *film.Service) *Service {
	repo := newRepository(pool)
	return &Service{pool: pool, repo: repo, filmService: filmService}
}

func (s *Service) GetById(ctx context.Context, id uuid.UUID) (*Duel, error) {
	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	WHERE films.id IN (
		SELECT film_a_id FROM duels WHERE id = $1
		UNION
		SELECT film_b_id FROM duels WHERE id = $1
	)`

	rows, err := s.pool.Query(ctx, query, id)
	if err != nil {
		log.Println(err)
		return nil, err
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
			return nil, err
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
		return nil, err
	}

	duel := Duel{
		Id:    id,
		FilmA: &films[0],
		FilmB: &films[1],
	}
	return &duel, nil
}

func (s *Service) GetPartialDuel(ctx context.Context, id uuid.UUID) (*Duel, error) {
	query := `
	SELECT id, rating
	FROM films
	WHERE films.id IN (
		SELECT film_a_id FROM duels WHERE id = $1
		UNION
		SELECT film_b_id FROM duels WHERE id = $1
	)`

	rows, err := s.pool.Query(ctx, query, id)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	var films [2]film.Film
	i := 0
	for rows.Next() {
		var film_id int
		var rating float64
		err = rows.Scan(&film_id, &rating)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		films[i] = film.Film{
			Id:     film_id,
			Rating: rating,
		}

		i++
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	duel := Duel{
		Id:    id,
		FilmA: &films[0],
		FilmB: &films[1],
	}
	return &duel, nil
}

func (s *Service) CreateDuel(ctx context.Context) (*Duel, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
	SELECT films.id, title, release_year, image_path, rating, directors.id, directors.name
	FROM films
	JOIN directors ON films.director_id = directors.id
	ORDER BY RANDOM() LIMIT 2`

	filRows, err := tx.Query(ctx, query)
	if err != nil {
		log.Println(err)
		return nil, err
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
			return nil, err
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
		return nil, err
	}
	if i < 2 {
		return nil, fmt.Errorf("not enough films for a duel: got %d", i)
	}

	query = "INSERT INTO duels (film_a_id, film_b_id) VALUES ($1, $2) RETURNING id"
	var duelId uuid.UUID
	err = tx.QueryRow(ctx, query, films[0].Id, films[1].Id).Scan(&duelId)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	duel := Duel{
		Id:    duelId,
		FilmA: &films[0],
		FilmB: &films[1],
	}

	return &duel, nil
}

func (s *Service) ComposeDuel(ctx context.Context, winnerId int) (*Duel, error) {
	// TODO:
	// this function makes 3 calls to the database
	// this should be optimized to be done in one

	winner, err := s.filmService.GetFilm(ctx, winnerId)
	if err != nil {
		return nil, err
	}

	filmB, err := s.filmService.GetRandomFilm(ctx)
	if err != nil {
		return nil, err
	}

	duelId, err := s.repo.Insert(ctx, winner, filmB)
	if err != nil {
		return nil, err
	}

	return &Duel{
		Id:    duelId,
		FilmA: winner,
		FilmB: filmB,
	}, nil
}
