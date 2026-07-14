package film

import (
	"context"
	"filmmash/internal/database"
)

type Service struct {
	repo      *repository
	txManager *database.TxManager
}

func NewService(repo *repository, txManager *database.TxManager) *Service {
	return &Service{
		repo:      repo,
		txManager: txManager,
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
