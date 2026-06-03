package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Director struct {
	Id   int
	Name string
}

type Film struct {
	Id        int
	Title     string
	Year      int
	Director  Director
	ImagePath string
	Rating    int
}

type Service struct {
	pool *pgxpool.Pool
}

func (s *Service) GetFilm(ctx context.Context, id int) (*Film, error) {
	query := "SELECT films.id, title, release_year, image_path, rating, " +
		"directors.id, directors.name FROM films JOIN directors ON films.director_id = directors.id " +
		"WHERE films.id = ($1)"

	rows, err := s.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var film_id, release_year, director_id, rating int
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
	return &film, nil
}

func (s *Service) GetDuel(ctx context.Context) (*[2]Film, error) {
	query := "SELECT films.id, title, release_year, image_path, rating, " +
		"directors.id, directors.name FROM films JOIN directors ON films.director_id = directors.id " +
		"ORDER BY RANDOM() LIMIT 2"

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var films [2]Film
	i := 0
	for rows.Next() {
		var film_id, release_year, director_id, rating int
		var title, image_path, director_name string
		err = rows.Scan(&film_id, &title, &release_year, &image_path, &rating, &director_id, &director_name)
		if err != nil {
			return nil, err
		}

		films[i] = Film{
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

		i++
	}
	return &films, nil
}
