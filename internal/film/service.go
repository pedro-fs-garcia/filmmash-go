package film

import (
	"context"
	"filmmash/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo      *repository
	txManager *database.TxManager
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo:      NewRepository(pool),
		txManager: database.NewTxManager(pool),
	}
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
