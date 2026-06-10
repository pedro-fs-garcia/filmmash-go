package vote

import (
	"context"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool        *pgxpool.Pool
	repo        *repository
	filmService *film.Service
	duelService *duel.Service
}

func NewService(
	pool *pgxpool.Pool,
	filmService *film.Service,
	duelService *duel.Service,
) *Service {
	repo := repository{
		pool: pool,
	}
	return &Service{
		pool:        pool,
		repo:        &repo,
		filmService: filmService,
		duelService: duelService,
	}
}

func (s *Service) RegisterVote(ctx context.Context, duelId uuid.UUID, winnerId int) (Vote, error) {
	ratings, err := s.duelService.GetDuelRatings(ctx, duelId)

	if err != nil {
		log.Println(err)
		return Vote{}, err
	}

	log.Println(ratings[0].Id, ratings[1].Id)

	var winner, loser duel.FilmRating
	switch winnerId {
	case ratings[0].Id:
		winner, loser = ratings[0], ratings[1]
	case ratings[1].Id:
		winner, loser = ratings[1], ratings[0]
	default:
		err = fmt.Errorf("winner does not belong to duel %s", duelId.String())
		log.Println(err)
		return Vote{}, err
	}

	// TODO: Use real current ratings from DB
	// TODO: Register new ratings on DB
	newWinnerRating, newLoserRating := film.CalculateRatings(winner.Rating, loser.Rating)

	vote := Vote{
		DuelId:            duelId,
		WinnerID:          winnerId,
		LoserId:           loser.Id,
		WinnerRatingAfter: newWinnerRating,
		LoserRatingAfter:  newLoserRating,
	}
	err = s.repo.InsertVote(ctx, &vote)
	if err != nil {
		log.Println(err)
		return Vote{}, err
	}
	return vote, nil
}
