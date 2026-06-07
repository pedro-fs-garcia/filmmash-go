package vote

import (
	"context"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"fmt"
	"log"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Vote struct {
	Id                uuid.UUID `db:"id"`
	DuelId            uuid.UUID `db:"duel_id"`
	WinnerID          int       `db:"winner_id"`
	LoserId           int       `db:"loser_id"`
	WinnerRatingAfter float64   `db:"winner_rating_after"`
	LoserRatingAfter  float64   `db:"loser_rating_after"`
	AppliedSeq        int64     `db:"applied_seq"`
}

type Service struct {
	pool        *pgxpool.Pool
	filmService *film.Service
	duelService *duel.Service
}

func NewService(pool *pgxpool.Pool, filmService *film.Service, duelService *duel.Service) *Service {
	return &Service{
		pool:        pool,
		filmService: filmService,
		duelService: duelService,
	}
}

func (s *Service) InsertVote(ctx context.Context, vote Vote) (*Vote, error) {
	query := `
	INSERT INTO votes (duel_id, winner_id, loser_id, winner_rating_after, loser_rating_after) 
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, duel_id, winner_id, loser_id, winner_rating_after, loser_rating_after, applied_seq`

	rows, err := s.pool.Query(ctx, query,
		vote.DuelId,
		vote.WinnerID,
		vote.LoserId,
		vote.WinnerRatingAfter,
		vote.LoserRatingAfter,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	v, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Vote])

	if err != nil {
		log.Println("Failed to convert db response to Vote struct", err)
		return nil, err
	}

	return &v, nil
}

func (s *Service) RegisterVote(ctx context.Context, duelId uuid.UUID, winnerId int) (*Vote, error) {
	duel, err := s.duelService.GetPartialDuel(ctx, duelId)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println(duel.FilmA.Id, duel.FilmB.Id)

	if winnerId != duel.FilmA.Id && winnerId != duel.FilmB.Id {
		err = fmt.Errorf("Winner does not belong to duel %s", duelId.String())
		log.Println(err)
		return nil, err
	}

	var loserId int
	var winnerRating, loserRating float64
	if winnerId == duel.FilmA.Id {
		loserId = duel.FilmB.Id
		loserRating = duel.FilmB.Rating
		winnerRating = duel.FilmA.Rating
	} else {
		loserId = duel.FilmA.Id
		loserRating = duel.FilmA.Rating
		winnerRating = duel.FilmB.Rating
	}

	var newWinnerRating, newLoserRating float64

	// TODO: Use real current ratings from DB
	// TODO: Register new ratings on DB
	CalculateRatings(winnerRating, loserRating, &newWinnerRating, &newLoserRating)

	vote, err := s.InsertVote(ctx, Vote{
		DuelId:            duelId,
		WinnerID:          winnerId,
		LoserId:           loserId,
		WinnerRatingAfter: newWinnerRating,
		LoserRatingAfter:  newLoserRating,
	})

	if err != nil {
		log.Println(err)
		return nil, err
	}
	return vote, nil
}

func CalculateRatings(winnerRating, loserRating float64, newWinnerRating, newLoserRating *float64) {
	const K = 20
	Ea := 1 / (1 + math.Pow(10, (winnerRating-loserRating)/400))
	Eb := 1 / (1 + math.Pow(10, (loserRating-winnerRating)/400))
	*newWinnerRating = winnerRating + K*(1-Ea)
	*newLoserRating = loserRating + K*(0-Eb)
}
