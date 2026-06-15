package film

import (
	"context"
	"filmmash/internal/database"
	"fmt"
	"log/slog"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	logger    *slog.Logger
	repo      *repository
	txManager *database.TxManager
}

func NewService(logger *slog.Logger, pool *pgxpool.Pool) *Service {
	return &Service{
		logger:    logger.With(slog.String("component", "Service")),
		repo:      NewRepository(logger, pool),
		txManager: database.NewTxManager(pool),
	}
}

func (s *Service) GetFilm(ctx context.Context, id int) (Film, error) {
	film, err := s.repo.GetFilm(ctx, id)
	if err != nil {
		return Film{}, err
	}
	return film, nil
}

func (s *Service) GetRandomFilm(ctx context.Context) (Film, error) {
	film, err := s.repo.GetRandomFilm(ctx)
	if err != nil {
		return Film{}, fmt.Errorf("Failed to get random film: %w", err)
	}
	return film, nil
}

func CalculateRatings(winnerRating, loserRating float64) (newWinnerRating, newLoserRating float64) {
	const K = 20
	Ea := 1 / (1 + math.Pow(10, (winnerRating-loserRating)/400))
	Eb := 1 / (1 + math.Pow(10, (loserRating-winnerRating)/400))
	newWinnerRating = winnerRating + K*(1-Ea)
	newLoserRating = loserRating + K*(0-Eb)
	return newWinnerRating, newLoserRating
}

func (s *Service) UpdateRatings(ctx context.Context, films []*FilmRating) error {
	return s.repo.UpdateRatings(ctx, films)
}

func (s *Service) CountTotal(ctx context.Context) (int, error) {
	return s.repo.CountTotal(ctx)
}
