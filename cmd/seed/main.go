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

	client := tmdb.NewClient("https://api.themoviedb.org/3", cfg.TmdbApitoken)
	totalTop := 0
	for page := 1; page <= 3; page++ {
		fmt.Printf("Downloading top rated movies from TMDB, page %d\n", page)
		movies, err := client.GetTopRated(int16(page))
		if err != nil {
			fmt.Println(err)
		}
		r := SaveMovies(ctx, pool, movies)
		fmt.Printf("%d new registers\n", r)
		totalTop += r
	}
	fmt.Printf("%d new top rated registers\n", totalTop)

	totalPopular := 0
	for page := 1; page <= 3; page++ {
		fmt.Printf("Downloading Popular movies from TMDB, page %d\n", page)
		movies, err := client.GetPopulars(int16(page))
		if err != nil {
			fmt.Println(err)
		}
		r := SaveMovies(ctx, pool, movies)
		fmt.Printf("%d new registers\n", r)
		totalPopular += r
	}
	fmt.Printf("%d new popular registers\n", totalPopular)

	fmt.Printf("%d Total new registers", totalTop+totalPopular)
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

		year, _, _ := strings.Cut(movie.ReleaseDate, "-")
		release_year, _ := strconv.Atoi(year)
		query := `
			INSERT INTO films (id, title, release_year, director_id, image_path, popularity, vote_average) 
			VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING
			`
		batch.Queue(query,
			movie.Id, movie.Title, release_year, movie.Director.Id,
			movie.PosterPath, movie.Popularity, movie.VoteAverage,
		)
	}

	result := pool.SendBatch(ctx, batch)

	defer func() {
		_ = result.Close()
	}()

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
