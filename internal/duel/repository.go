package duel

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"filmmash/internal/film"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

func (r *repository) queries(ctx context.Context) *dbgen.Queries {
	return dbgen.New(database.ExtractTx(ctx, r.pool))
}

func filmFromRow(row dbgen.GetDuelFilmsRow) film.Film {
	f := film.Film{
		Id:     int(row.ID),
		Title:  row.Title,
		Year:   int(row.ReleaseYear),
		Rating: row.Rating,
		Director: film.Director{
			Id:   int(row.DirectorID),
			Name: row.DirectorName,
		},
	}
	if row.ImagePath != nil {
		f.ImagePath = *row.ImagePath
	}
	return f
}

func (r *repository) Insert(ctx context.Context, duel *Duel) error {
	id, err := r.queries(ctx).InsertDuel(ctx, dbgen.InsertDuelParams{
		FilmAID: int32(duel.FilmA.Id),
		FilmBID: int32(duel.FilmB.Id),
	})
	if err != nil {
		return parseDBError("inserting duel", err)
	}
	duel.Id = id
	return nil
}

func (r *repository) GetById(ctx context.Context, id uuid.UUID) (Duel, error) {
	rows, err := r.queries(ctx).GetDuelFilms(ctx, id)
	if err != nil {
		return Duel{}, parseDBError("querying duel by id", err)
	}
	if len(rows) < 2 {
		return Duel{}, fmt.Errorf("Duel %v references missing films: %w", id, ErrNotEnoughFilms)
	}

	var films [2]film.Film
	for i := range 2 {
		films[i] = filmFromRow(rows[i])
	}
	return Duel{
		Id:    id,
		FilmA: &films[0],
		FilmB: &films[1],
	}, nil
}

func (r *repository) GetDuelRatingsForUpdate(ctx context.Context, id uuid.UUID) ([2]film.FilmRating, error) {
	rows, err := r.queries(ctx).GetDuelRatingsForUpdate(ctx, id)
	if err != nil {
		return [2]film.FilmRating{}, parseDBError(fmt.Sprintf("querying Duel(id: %v) Ratings", id), err)
	}
	if len(rows) < 2 {
		return [2]film.FilmRating{}, fmt.Errorf("Duel %v references missing films: %w", id, ErrNotEnoughFilms)
	}

	var ratings [2]film.FilmRating
	for i := range 2 {
		ratings[i] = film.FilmRating{
			Id:        int(rows[i].ID),
			Rating:    rows[i].Rating,
			DuelCount: rows[i].DuelCount,
		}
	}
	return ratings, nil
}

func (r *repository) SelectRandomFilms(ctx context.Context) ([2]film.Film, error) {
	rows, err := r.queries(ctx).SelectRandomFilms(ctx)
	if err != nil {
		return [2]film.Film{}, parseDBError("querying random films", err)
	}
	if len(rows) < 2 {
		return [2]film.Film{}, fmt.Errorf("not enough films for a duel (got %d): %w", len(rows), ErrNotEnoughFilms)
	}

	var films [2]film.Film
	for i := range 2 {
		films[i] = filmFromRow(dbgen.GetDuelFilmsRow(rows[i]))
	}
	return films, nil
}

func (r *repository) FindCandidates(
	ctx context.Context,
	id int32, minCandidates,
	maxCandidates int16,
	ratingWindow float64,
) ([]Candidate, error) {
	rows, err := r.queries(ctx).FindCandidates(ctx, dbgen.FindCandidatesParams{
		WinnerID:      id,
		MinCandidates: int32(minCandidates),
		MaxCandidates: int32(maxCandidates),
		RatingWindow:  ratingWindow,
	})
	if err != nil {
		return nil, database.ParseDBError(fmt.Sprintf("querying for candidates to film %d", id), err)
	}

	candidates := make([]Candidate, len(rows))
	for i, r := range rows {
		candidates[i] = Candidate{
			Id:         r.ID,
			Popularity: r.Popularity,
			DuelCount:  r.DuelCount,
			CreatedAt:  r.CreatedAt,
		}
	}
	return candidates, nil
}

func (r *repository) CreateDuel(ctx context.Context, filmAId, filmBId int32) (Duel, error) {
	row, err := r.queries(ctx).CreateDuel(ctx, dbgen.CreateDuelParams{
		FilmAID: filmAId,
		FilmBID: filmBId,
	})
	if err != nil {
		return Duel{}, database.ParseDBError(fmt.Sprintf("creating duel for films %d, %d", filmAId, filmBId), err)
	}
	return duelFromRow(row), nil
}

func duelFromRow(row dbgen.CreateDuelRow) Duel {
	filmA := film.Film{
		Id:     int(row.FilmAID),
		Title:  row.FilmATitle,
		Year:   int(row.FilmAReleaseYear),
		Rating: row.FilmARating,
		Director: film.Director{
			Id:   int(row.DirectorAID),
			Name: row.DirectorAName,
		},
	}
	if row.FilmAImagePath != nil {
		filmA.ImagePath = *row.FilmAImagePath
	}

	filmB := film.Film{
		Id:     int(row.FilmBID),
		Title:  row.FilmBTitle,
		Year:   int(row.FilmBReleaseYear),
		Rating: row.FilmBRating,
		Director: film.Director{
			Id:   int(row.DirectorBID),
			Name: row.DirectorBName,
		},
	}
	if row.FilmBImagePath != nil {
		filmB.ImagePath = *row.FilmBImagePath
	}

	return Duel{
		Id:    row.DuelID,
		FilmA: &filmA,
		FilmB: &filmB,
	}
}

func (r *repository) ComposeDuel(
	ctx context.Context,
	winnerId int,
	popularityWeight float64,
	ratingWindow int32,
) (Duel, error) {
	row, err := r.queries(ctx).ComposeDuel(ctx, dbgen.ComposeDuelParams{
		WinnerID:         int32(winnerId),
		RatingWindow:     float64(ratingWindow),
		PopularityWeight: popularityWeight,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Duel{}, fmt.Errorf("no opponent film available for winner(id: %v): %w", winnerId, ErrNotEnoughFilms)
		}
		return Duel{}, parseDBError(fmt.Sprintf("composing duel for winner %d", winnerId), err)
	}
	return duelFromRow(dbgen.CreateDuelRow(row)), nil
}

func (r *repository) CountPending(ctx context.Context) (int, error) {
	n, err := r.queries(ctx).CountPendingDuels(ctx)
	if err != nil {
		return 0, parseDBError("counting pending duels", err)
	}
	return int(n), nil
}

func (r *repository) ComposeWeightedDuel(
	ctx context.Context,
	id int32,
	minCandidates, maxCandidates int16,
	ratingWindow float64,
	popularityWeight float64,
	newFilmBoost float64,
	newFilmDuelThreshold int16,
) (Duel, error) {
	row, err := r.queries(ctx).ComposeWeightedDuel(ctx, dbgen.ComposeWeightedDuelParams{
		WinnerID:             id,
		MinCandidates:        int32(minCandidates),
		MaxCandidates:        int32(maxCandidates),
		RatingWindow:         ratingWindow,
		PopularityWeight:     popularityWeight,
		NewFilmBoost:         newFilmBoost,
		NewFilmDuelThreshold: int32(newFilmDuelThreshold),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Duel{}, fmt.Errorf("no opponent film available for winner(id: %v): %w", id, ErrNotEnoughFilms)
		}
		return Duel{}, parseDBError(fmt.Sprintf("composing weighted duel for winner %d", id), err)
	}
	return duelFromRow(dbgen.CreateDuelRow(row)), nil
}
