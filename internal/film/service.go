package film

import (
	"context"
	"filmmash/internal/database"
	"math"

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
	if len(films) == 0 {
		return PaginatedResponse{}, nil
	}
	last := films[len(films)-1]
	return PaginatedResponse{
		Films: films,
		Next: PaginationParameters{
			Size:           pars.Size,
			LastSeenId:     last.Id,
			LastSeenRating: last.Rating,
		},
	}, nil
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

func CalculateRatings(winnerRating, loserRating float64) (newWinnerRating, newLoserRating float64) {
	const K = 20
	Ea := 1 / (1 + math.Pow(10, (winnerRating-loserRating)/400))
	Eb := 1 / (1 + math.Pow(10, (loserRating-winnerRating)/400))
	newWinnerRating = winnerRating + K*(1-Ea)
	newLoserRating = loserRating + K*(0-Eb)
	return newWinnerRating, newLoserRating
}
