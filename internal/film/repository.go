package film

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

func (r *repository) queries(ctx context.Context) *dbgen.Queries {
	return dbgen.New(database.ExtractTx(ctx, r.pool))
}

func filmFromRow(row dbgen.GetFilmRow) Film {
	f := Film{
		Id:     int(row.ID),
		Title:  row.Title,
		Year:   int(row.ReleaseYear),
		Rating: row.Rating,
		Director: Director{
			Id:   int(row.DirectorID),
			Name: row.DirectorName,
		},
	}
	if row.ImagePath != nil {
		f.ImagePath = *row.ImagePath
	}
	return f
}

func (r *repository) InsertFilm(ctx context.Context, f *Film) error {
	if f.Rating == 0 {
		f.Rating = StdRating
	}
	id, err := r.queries(ctx).InsertFilm(ctx, dbgen.InsertFilmParams{
		ID:           int32(f.Id),
		Title:        f.Title,
		ReleaseYear:  int16(f.Year),
		ImagePath:    &f.ImagePath,
		Rating:       f.Rating,
		DirectorID:   int32(f.Director.Id),
		DirectorName: f.Director.Name,
	})
	if err != nil {
		return parseDBError("inserting film", err)
	}
	f.Id = int(id)
	return nil
}

func (r *repository) InsertDirector(ctx context.Context, d *Director) error {
	id, err := r.queries(ctx).InsertDirector(ctx, dbgen.InsertDirectorParams{
		ID:   int32(d.Id),
		Name: d.Name,
	})
	if err != nil {
		return parseDBError(
			fmt.Sprintf("inserting director (id: %v)", d.Id),
			err,
		)
	}
	d.Id = int(id)
	return nil
}

func (r *repository) GetDirector(ctx context.Context, id int) (Director, error) {
	row, err := r.queries(ctx).GetDirector(ctx, int32(id))
	if err != nil {
		return Director{}, parseDBError(
			fmt.Sprintf("getting director by id (id: %d)", id),
			err,
		)
	}
	return Director{Id: int(row.ID), Name: row.Name}, nil
}

func (r *repository) GetFilm(ctx context.Context, id int) (Film, error) {
	row, err := r.queries(ctx).GetFilm(ctx, int32(id))
	if err != nil {
		return Film{}, parseDBError(
			fmt.Sprintf("getting film by id (id: %d)", id),
			err,
		)
	}
	return filmFromRow(row), nil
}

func (r *repository) GetRandomFilm(ctx context.Context) (Film, error) {
	row, err := r.queries(ctx).GetRandomFilm(ctx)
	if err != nil {
		return Film{}, parseDBError(
			"getting random film",
			err,
		)
	}
	return filmFromRow(dbgen.GetFilmRow(row)), nil
}

func (r *repository) GetFilmsPaginatedByRating(ctx context.Context, pars PaginationParameters) ([]Film, error) {
	params := dbgen.ListFilmsByRatingParams{
		Limit: int32(pars.Size),
	}
	if pars.LastSeenId != 0 {
		lastSeenId := int32(pars.LastSeenId)
		lastSeenRating := pars.LastSeenRating
		params.LastSeenID = &lastSeenId
		params.LastSeenRating = &lastSeenRating
	}

	rows, err := r.queries(ctx).ListFilmsByRating(ctx, params)
	if err != nil {
		return nil, parseDBError("querying paginated films", err)
	}

	var films []Film
	for _, row := range rows {
		films = append(films, filmFromRow(dbgen.GetFilmRow(row)))
	}
	return films, nil
}

func (r *repository) SearchFilmByName(ctx context.Context, search string) ([]Film, error) {
	rows, err := r.queries(ctx).SearchFilmsByTitle(ctx, search)
	if err != nil {
		return nil, parseDBError("querying films by name", err)
	}

	var films []Film
	for _, row := range rows {
		films = append(films, filmFromRow(dbgen.GetFilmRow(row)))
	}
	return films, nil
}

func (r *repository) UpdateRating(ctx context.Context, f *Film) error {
	affected, err := r.queries(ctx).UpdateFilmRating(ctx, dbgen.UpdateFilmRatingParams{
		Rating: f.Rating,
		ID:     int32(f.Id),
	})
	if err != nil {
		return parseDBError(fmt.Sprintf("updating rating for film(id: %d)", f.Id), err)
	}
	if affected == 0 {
		return fmt.Errorf("film not found to update rating (id: %d): %w", f.Id, ErrNotFound)
	}
	return nil
}

func (r *repository) UpdateRatings(ctx context.Context, films []*FilmRating) error {
	if len(films) == 0 {
		return nil
	}

	params := make([]dbgen.UpdateFilmRatingsParams, len(films))
	for i := range films {
		params[i] = dbgen.UpdateFilmRatingsParams{
			Rating: films[i].Rating,
			ID:     int32(films[i].Id),
		}
	}

	var batchErr error
	r.queries(ctx).UpdateFilmRatings(ctx, params).QueryRow(func(_ int, _ int32, err error) {
		if err != nil && batchErr == nil {
			if errors.Is(err, pgx.ErrNoRows) {
				batchErr = fmt.Errorf("no register affected on batch update: %w", ErrNotFound)
				return
			}
			batchErr = parseDBError("updating ratings for film batch", err)
		}
	})
	return batchErr
}

func (r *repository) CountTotal(ctx context.Context) (int64, error) {
	n, err := r.queries(ctx).CountFilms(ctx)
	if err != nil {
		return 0, parseDBError("counting films", err)
	}
	return n, nil
}

func (r *repository) IdsInCatalogue(ctx context.Context, ids []int32) ([]int32, error) {
	idList, err := r.queries(ctx).IdsInFilms(ctx, ids)
	if err != nil {
		return nil, database.ParseDBError("querying existing films", err)
	}
	return idList, nil
}
