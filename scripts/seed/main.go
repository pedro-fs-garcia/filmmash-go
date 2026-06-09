package main

import (
	"context"
	"filmmash/internal/config"
	"filmmash/internal/database"
	"filmmash/internal/tmdb"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PgDbName,
	)

	pool, err := database.Pool(ctx, dsn)
	if err != nil {
		panic(err)
	}

	client := tmdb.NewTmdbClient()
	total := 0
	for page := 1; page <= 3; page++ {
		fmt.Printf("Downloading movies from tmdb, page %d", page)
		movies, err := client.GetTopRated(page)
		if err != nil {
			fmt.Println(err)
		}
		r := SaveMovies(ctx, pool, movies)
		fmt.Printf("%d new registers\n", r)
		total += r
	}
	fmt.Printf("%d Total new registers\n", total)
}

func SaveMovies(ctx context.Context, pool *pgxpool.Pool, movies []tmdb.Movie) int {
	fmt.Printf("Saving movies to database\n")

	batch := &pgx.Batch{}

	for i := range movies {
		movie := movies[i]
		batch.Queue(
			"INSERT INTO directors (id, name) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			movie.Director.Id, movie.Director.Name,
		)
		// fmt.Printf("Saving director (%d, %s)", movie.Director.Id, movie.Director.Name)

		year, _, _ := strings.Cut(movie.ReleaseDate, "-")
		release_year, _ := strconv.Atoi(year)
		batch.Queue(
			"INSERT INTO films (id, title, release_year, director_id, image_path) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING",
			movie.Id, movie.Title, release_year, movie.Director.Id, movie.PosterPath,
		)
		// fmt.Printf("Saving film (%s)", movie.Title)
	}

	result := pool.SendBatch(ctx, batch)

	defer result.Close()

	total := 0
	for range movies {
		tag, err := result.Exec()
		if err != nil {
			fmt.Printf("Error reading results")
		}
		total += int(tag.RowsAffected())
	}
	return total

}
