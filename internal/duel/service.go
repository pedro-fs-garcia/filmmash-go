package duel

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	metrics     *metrics.DuelMetrics
	repo        *repository
	txManager   *database.TxManager
	filmService *film.Service
}

func NewService(pool *pgxpool.Pool, filmService *film.Service) *Service {
	repo := NewRepository(pool)
	txManager := database.NewTxManager(pool)
	return &Service{repo: repo, txManager: txManager, filmService: filmService}
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
			log.Println(err)
			return err
		}

		duel = Duel{
			FilmA: &films[0],
			FilmB: &films[1],
		}

		if err = s.repo.Insert(txCtx, &duel); err != nil {
			log.Println(err)
		}
		return err
	})
	if err != nil {
		return Duel{}, err
	}

	s.metrics.DuelCreated()
	return duel, nil
}

func (s *Service) ComposeDuel(ctx context.Context, winnerId int) (Duel, error) {
	duel, err := s.repo.ComposeDuel(ctx, winnerId)
	if err != nil {
		log.Println(err)
		return Duel{}, nil
	}
	s.metrics.DuelCreated()
	return duel, nil
}

func (s *Service) CountPending(ctx context.Context) (int, error) {
	return s.repo.CountPending(ctx)
}
