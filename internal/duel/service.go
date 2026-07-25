package duel

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"math/rand/v2"

	"github.com/google/uuid"
)

type ServiceConfig struct {
	MinCandidates        int16
	MaxCandidates        int16
	RatingWindow         int32
	PopularityWeight     float64
	NewFilmBoost         float64
	NewFilmDuelThreshold int16
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

func (s *Service) candidateWeight(c Candidate) float64 {
	w := 1.0 + c.Popularity*s.cfg.PopularityWeight
	if s.cfg.NewFilmBoost > 1 && c.DuelCount < int32(s.cfg.NewFilmDuelThreshold) {
		w *= s.cfg.NewFilmBoost
	}
	return w
}

func DrawFromCandidates(candidates []Candidate, weight func(Candidate) float64) (int32, bool) {
	if len(candidates) == 0 {
		return 0, false
	}
	total := 0.0
	for _, c := range candidates {
		total += weight(c)
	}
	r := rand.Float64() * total
	for _, c := range candidates {
		r -= weight(c)
		if r < 0 {
			return c.Id, true
		}
	}
	return candidates[len(candidates)-1].Id, true
}

func (s *Service) DuelFromFilm(ctx context.Context, filmId int32) (Duel, error) {
	var duel Duel
	err := s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		candidates, err := s.repo.FindCandidates(txCtx, filmId, s.cfg.MinCandidates, s.cfg.MaxCandidates, float64(s.cfg.RatingWindow))
		if err != nil {
			return err
		}

		fid, ok := DrawFromCandidates(candidates, s.candidateWeight)
		if !ok {
			return ErrNoCandidates
		}
		duel, err = s.repo.CreateDuel(txCtx, filmId, fid)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Duel{}, err
	}
	s.metrics.DuelCreated()
	return duel, nil
}
