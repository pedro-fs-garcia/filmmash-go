package duel

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/metrics"

	"github.com/google/uuid"
)

type ServiceConfig struct {
	PopularityWeight float64
	RatingWindow     int32
}

type Service struct {
	metrics     *metrics.DuelMetrics
	repo        *repository
	txManager   *database.TxManager
	filmService *film.Service
	cfg         ServiceConfig
}

func NewService(metrics *metrics.DuelMetrics, repo *repository, txm *database.TxManager, filmService *film.Service, cfg ServiceConfig) *Service {
	return &Service{
		metrics:     metrics,
		repo:        repo,
		txManager:   txm,
		filmService: filmService,
		cfg:         cfg,
	}
}

func (s *Service) GetById(ctx context.Context, id uuid.UUID) (Duel, error) {
	return s.repo.GetById(ctx, id)
}

func (s *Service) GetDuelRatings(ctx context.Context, id uuid.UUID) ([2]film.FilmRating, error) {
	return s.repo.GetDuelRatingsForUpdate(ctx, id)
}

func (s *Service) CreateRandomDuel(ctx context.Context) (Duel, error) {
	var duel Duel
	err := s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		films, err := s.repo.SelectRandomFilms(txCtx)
		if err != nil {
			return err
		}

		duel = Duel{
			FilmA: &films[0],
			FilmB: &films[1],
		}
		return s.repo.Insert(txCtx, &duel)
	})
	if err != nil {
		return Duel{}, err
	}

	s.metrics.DuelCreated()
	return duel, nil
}

func (s *Service) ComposeDuel(ctx context.Context, winnerId int) (Duel, error) {
	duel, err := s.repo.ComposeDuel(ctx, winnerId, s.cfg.PopularityWeight, s.cfg.RatingWindow)
	if err != nil {
		return Duel{}, err
	}
	s.metrics.DuelCreated()
	return duel, nil
}

func (s *Service) CountPending(ctx context.Context) (int, error) {
	return s.repo.CountPending(ctx)
}
