package film

import (
	"context"
	"filmmash/internal/database"
	"fmt"
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
	film, err := s.repo.GetFilm(ctx, id)
	if err != nil {
		return Film{}, fmt.Errorf("[film.Service.GetFilm] Error to get film by id: %w", err)
	}
	return film, nil
}

func (s *Service) GetRandomFilm(ctx context.Context) (Film, error) {
	film, err := s.repo.GetRandomFilm(ctx)
	if err != nil {
		return Film{}, fmt.Errorf("[film.Service.GetRandomFilm] Failed to get random film: %w", err)
	}
	return film, nil
}

func (s *Service) UpdateRatings(ctx context.Context, films []*FilmRating) error {
	if len(films) == 0 {
		return nil
	}
	err := s.repo.UpdateRatings(ctx, films)
	if err != nil {
		return fmt.Errorf("[film.Service.UpdateRatings] Failure in update batch usecase: %w", err)
	}
	return nil
}

func (s *Service) CountTotal(ctx context.Context) (int64, error) {
	count, err := s.repo.CountTotal(ctx)
	if err != nil {
		return 0, fmt.Errorf("[film.Service.CountTotal] Error counting total films: %w", err)
	}
	return count, nil
}

func CalculateRatings(winnerRating, loserRating float64) (newWinnerRating, newLoserRating float64) {
	const K = 20
	Ea := 1 / (1 + math.Pow(10, (winnerRating-loserRating)/400))
	Eb := 1 / (1 + math.Pow(10, (loserRating-winnerRating)/400))
	newWinnerRating = winnerRating + K*(1-Ea)
	newLoserRating = loserRating + K*(0-Eb)
	return newWinnerRating, newLoserRating
}
