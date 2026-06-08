package duel

import (
	"context"
	"filmmash/internal/film"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool        *pgxpool.Pool
	repo        *repository
	filmService *film.Service
}

func NewService(pool *pgxpool.Pool, filmService *film.Service) *Service {
	repo := newRepository(pool)
	return &Service{pool: pool, repo: repo, filmService: filmService}
}

func (s *Service) GetById(ctx context.Context, id uuid.UUID) (Duel, error) {
	return s.repo.GetById(ctx, id)
}

func (s *Service) GetDuelRatings(ctx context.Context, id uuid.UUID) ([2]FilmRating, error) {
	return s.repo.GetDuelRatings(ctx, id)
}

func (s *Service) CreateRandomDuel(ctx context.Context) (Duel, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Println(err)
		return Duel{}, err
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)
	films, err := txRepo.SelectRandomFilms(ctx)
	if err != nil {
		return Duel{}, err
	}

	duel := Duel{
		FilmA: &films[0],
		FilmB: &films[1],
	}

	err = txRepo.Insert(ctx, &duel)
	if err != nil {
		log.Println(err)
		return Duel{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Println(err)
		return Duel{}, err
	}

	return duel, nil
}

func (s *Service) ComposeDuel(ctx context.Context, winnerId int) (Duel, error) {
	// TODO:
	// this function makes 3 calls to the database
	// this should be optimized to be done in one

	winner, err := s.filmService.GetFilm(ctx, winnerId)
	if err != nil {
		return Duel{}, err
	}

	filmB, err := s.filmService.GetRandomFilm(ctx)
	if err != nil {
		return Duel{}, err
	}
	duel := Duel{
		FilmA: &winner,
		FilmB: &filmB,
	}
	err = s.repo.Insert(ctx, &duel)
	if err != nil {
		return Duel{}, err
	}
	return duel, nil
}
