package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Director struct {
}

type Film struct {
	Id         int
	Title      string
	Year       int
	DirectorId int
	ImagePath  string
	Rating     int
}

type Service struct {
	pool *pgxpool.Pool
}

func (s *Service) GetFilm(ctx context.Context, id int) (*Film, error) {

	rows, err := s.pool.Query(ctx, "SELECT title, release_year, director_id, image_path, rating FROM films WHERE id = ($1)", id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var release_year, director_id, rating int
	var title, image_path string
	rows.Next()
	err = rows.Scan(&title, &release_year, &director_id, &image_path, &rating)

	film := Film{
		Id:         id,
		Title:      title,
		Year:       release_year,
		DirectorId: director_id,
		ImagePath:  image_path,
		Rating:     rating,
	}
	return &film, nil
}
