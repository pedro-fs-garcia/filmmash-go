package film

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/tmdb"
	"fmt"
	"log/slog"
	"time"
)

type Service struct {
	logger    *slog.Logger
	repo      *repository
	txManager *database.TxManager
	client    *tmdb.Client
}

func NewService(logger *slog.Logger, repo *repository, txManager *database.TxManager, client *tmdb.Client) *Service {
	return &Service{
		logger:    logger,
		repo:      repo,
		txManager: txManager,
		client:    client,
	}
}

func (s *Service) InsertFilm(ctx context.Context, f *Film) error {
	return s.repo.InsertFilm(ctx, f)
}

func (s *Service) GetFilm(ctx context.Context, id int) (Film, error) {
	return s.repo.GetFilm(ctx, id)
}

func (s *Service) GetRandomFilm(ctx context.Context) (Film, error) {
	return s.repo.GetRandomFilm(ctx)
}

func (s *Service) GetFilmsPaginatedByRating(ctx context.Context, pars PaginationParameters) (PaginatedResponse, error) {
	films, err := s.repo.GetFilmsPaginatedByRating(ctx, pars)
	if err != nil {
		return PaginatedResponse{}, err
	}
	return ToPaginatedResponse(pars.Size, films), nil
}

func (s *Service) SearchFilmByName(ctx context.Context, search string) ([]Film, error) {
	return s.repo.SearchFilmByName(ctx, search)
}

func (s *Service) UpdateRatings(ctx context.Context, films []*FilmRating) error {
	if len(films) == 0 {
		return nil
	}
	return s.repo.UpdateRatings(ctx, films)
}

func (s *Service) CountTotal(ctx context.Context) (int64, error) {
	return s.repo.CountTotal(ctx)
}

func (s *Service) IdsInCatalogue(ctx context.Context, ids []int32) (map[int32]bool, error) {
	idList, err := s.repo.IdsInCatalogue(ctx, ids)
	if err != nil {
		return nil, err
	}
	existing := make(map[int32]bool, len(idList))
	for _, id := range idList {
		existing[id] = true
	}

	return existing, nil
}

func (s *Service) InsertFilmBatch(ctx context.Context, films []Film) (int32, error) {
	return s.repo.InsertFilmBatch(ctx, films)
}

func FetchFilms(
	ctx context.Context,
	logger *slog.Logger,
	fetchFunc func(page int16) ([]tmdb.Movie, error),
	page int16,
) ([]Film, error) {
	logger = logger.With("method", "FetchFilms")
	movies, err := fetchFunc(page)
	if err != nil {
		return nil, err
	}

	films := []Film{}
	for _, m := range movies {
		f, err := TMDBMovieToFilm(m)
		if err != nil {
			logger.ErrorContext(ctx, "parsing tmdb.Movie to film.Film",
				slog.Int("tmdb_id", m.Id), slog.String("error", err.Error()))
			continue
		}
		films = append(films, f)
	}
	return films, err
}

func (s *Service) SyncFilms(ctx context.Context, fetchFunc func(page int16) ([]tmdb.Movie, error)) {
	log := s.logger.With("method", "SyncFilms")
	var p int16
	for p = 1; p <= 3; p++ {
		films, err := FetchFilms(ctx, s.logger, fetchFunc, p)
		if err != nil {
			log.ErrorContext(ctx, "fetching films from tmdb films", slog.String("error", err.Error()))
			continue
		}
		t, err := s.InsertFilmBatch(ctx, films)
		if err != nil {
			log.ErrorContext(ctx, "updating films catalog from tmdb films", slog.String("error", err.Error()))
		} else {
			log.InfoContext(ctx, fmt.Sprintf("updated films catalog with %d films from tmdb", t))
		}
	}
}

func (s *Service) InsertFilmsJob(
	ctx context.Context,
	duration time.Duration,
	fetchFunc func(page int16) ([]tmdb.Movie, error),
) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	s.SyncFilms(ctx, fetchFunc)

	for {
		select {
		case <-ticker.C:
			s.SyncFilms(ctx, fetchFunc)
		case <-ctx.Done():
			return
		}
	}
}
