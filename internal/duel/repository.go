package duel

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	err := q.QueryRow(ctx, query, duel.FilmA.Id, duel.FilmB.Id).Scan(&duel.Id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf(
				"[duel.repository.Insert] conflict inserting duel: %w", ErrDuplicateEntry,
			)
		}
		return fmt.Errorf("[duel.repository.Insert] Failed to insert duel: %w", err)
	}
	return nil
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
		return Duel{}, fmt.Errorf("[duel.repository.GetById] Error querying Duel(id: %v): %w", id, err)
	}
	defer rows.Close()

	var films [2]film.Film
	i := 0
	for rows.Next() {
		var f film.Film
		err = rows.Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
		if err != nil {
			return Duel{}, fmt.Errorf("[duel.repository.GetById] Internal database error to get Duel(id: %v): %w", id, err)
		}

		films[i] = f
		i++
	}

	if err = rows.Err(); err != nil {
		return Duel{}, fmt.Errorf("[duel.repository.GetById] iterating rows for Duel(id: %v): %w", id, err)
	}
	if i < 2 {
		return Duel{}, fmt.Errorf("[duel.repository.GetById] Could not find Duel(id: %v): %w", id, ErrNotFound)
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
		return [2]film.FilmRating{}, fmt.Errorf("[duel.repository.GetDuelRatingsForUpdate] Error querying Duel(id: %v) Ratings: %w", id, err)
	}
	defer rows.Close()

	var ratings [2]film.FilmRating
	i := 0
	for rows.Next() {
		var fr film.FilmRating
		err = rows.Scan(&fr.Id, &fr.Rating)
		if err != nil {
			return [2]film.FilmRating{}, fmt.Errorf("[duel.repository.GetDuelRatingsForUpdate] Internal database error to get Duel(id: %v) Ratings: %w", id, err)
		}
		ratings[i] = fr
		i++
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("[duel.repository.GetDuelRatingsForUpdate] iterating rows for Duel(id: %v) Ratings: %w", id, err)
		return [2]film.FilmRating{}, err
	}
	if i < 2 {
		return [2]film.FilmRating{}, fmt.Errorf("[duel.repository.GetDuelRatingsForUpdate] duel %s does not have two films: %w", id, ErrInvalidDuel)
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
		return [2]film.Film{}, fmt.Errorf("[duel.repository.SelectRandomFilms] Error querying for random films: %w", err)
	}
	defer filRows.Close()

	var films [2]film.Film
	i := 0
	for filRows.Next() {
		var f film.Film
		err = filRows.Scan(&f.Id, &f.Title, &f.Year, &f.ImagePath, &f.Rating, &f.Director.Id, &f.Director.Name)
		if err != nil {
			return [2]film.Film{}, fmt.Errorf("[duel.repository.SelectRandomFilms] Internal database error to get random films: %w", err)
		}
		films[i] = f
		i++
	}
	if err = filRows.Err(); err != nil {
		return [2]film.Film{}, fmt.Errorf("[duel.repository.SelectRandomFilms] iterating rows for random films: %w", err)
	}
	if i < 2 {
		return [2]film.Film{}, fmt.Errorf("[duel.repository.SelectRandomFilms] not enough films for a duel: got %d: %w", i, ErrNotEnoughFilms)
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

	d := Duel{FilmA: &film.Film{}, FilmB: &film.Film{}}
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query, winnerId).Scan(
		&d.Id,
		&d.FilmA.Id, &d.FilmA.Title, &d.FilmA.Year, &d.FilmA.ImagePath, &d.FilmA.Rating, &d.FilmA.Director.Id, &d.FilmA.Director.Name,
		&d.FilmB.Id, &d.FilmB.Title, &d.FilmB.Year, &d.FilmB.ImagePath, &d.FilmB.Rating, &d.FilmB.Director.Id, &d.FilmB.Director.Name,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Duel{}, fmt.Errorf("[duel.repository.ComposeDuel] no opponent film available for winner(id: %v): %w", winnerId, ErrNotEnoughFilms)
		}
		return Duel{}, fmt.Errorf("[duel.repository.ComposeDuel] composing duel for winner(id: %v): %w", winnerId, err)
	}

	return d, nil
}

func (r *repository) CountPending(ctx context.Context) (int, error) {
	query := `
	SELECT COUNT(*) FROM duels d
	WHERE NOT EXISTS (SELECT 1 FROM votes v WHERE v.duel_id = d.id)
	`
	var n int
	q := database.ExtractTx(ctx, r.pool)
	err := q.QueryRow(ctx, query).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("[duel.repository.CountPending] counting pending duels: %w", err)
	}
	return n, nil
}
