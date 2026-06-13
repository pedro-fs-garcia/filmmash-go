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
	repo := repository{pool: pool}
	tm := database.NewTxManager(pool)
	return &Service{repo: &repo, txManager: tm}
}

func (s *Service) GetFilm(ctx context.Context, id int) (Film, error) {
	return s.repo.GetFilm(ctx, id)
}

func (s *Service) GetRandomFilm(ctx context.Context) (Film, error) {
	return s.repo.GetRandomFilm(ctx)
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
