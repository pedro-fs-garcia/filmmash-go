package vote

import (
	"context"
	"filmmash/internal/auth"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

type RegisterVoteUC struct {
	metrics     *metrics.VoteMetrics
	voteRepo    *repository
	txManager   *database.TxManager
	filmService *film.Service
	duelService *duel.Service
}

func NewRegisterVoteUC(metrics *metrics.VoteMetrics, repo *repository, txm *database.TxManager, filmService *film.Service, duelService *duel.Service) *RegisterVoteUC {
	return &RegisterVoteUC{
		metrics:     metrics,
		voteRepo:    repo,
		txManager:   txm,
		filmService: filmService,
		duelService: duelService,
	}
}

func (uc *RegisterVoteUC) RegisterVote(ctx context.Context, duelId uuid.UUID, winnerId int) (Vote, error) {
	var vote Vote
	var userId *uuid.UUID
	session, ok := auth.SessionFromContext(ctx)
	if ok {
		userId = &session.UserID
	}

	err := uc.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		ratings, err := uc.duelService.GetDuelRatings(txCtx, duelId)
		if err != nil {
			return err
		}

		var winner, loser film.FilmRating
		switch winnerId {
		case ratings[0].Id:
			winner, loser = ratings[0], ratings[1]
		case ratings[1].Id:
			winner, loser = ratings[1], ratings[0]
		default:
			return fmt.Errorf("[vote.RegisterVoteUC.RegisterVote] film(id: %v) does not belong to duel(id: %v): %w", winnerId, duelId, ErrFilmNotInDuel)
		}

		newWinnerRating, newLoserRating := film.CalculateRatings(
			winner.Rating,
			loser.Rating,
			film.FilmKFromDuelCount(winner.DuelCount),
			film.FilmKFromDuelCount(loser.DuelCount),
		)
		vote = Vote{
			DuelId:            duelId,
			UserID:            userId,
			WinnerID:          winnerId,
			LoserId:           loser.Id,
			WinnerRatingAfter: newWinnerRating,
			LoserRatingAfter:  newLoserRating,
		}
		err = uc.voteRepo.InsertVote(txCtx, &vote)
		if err != nil {
			return err
		}

		filmRatings := []*film.FilmRating{
			{Id: winnerId, Rating: newWinnerRating},
			{Id: loser.Id, Rating: newLoserRating},
		}
		return uc.filmService.UpdateRatings(txCtx, filmRatings)
	})
	if err != nil {
		return Vote{}, fmt.Errorf("[vote.RegisterVoteUC.RegisterVote] %w", err)
	}

	uc.metrics.VoteRegistered(strconv.Itoa(vote.WinnerID))
	return vote, nil
}

func (uc *RegisterVoteUC) NewDuelFromWinner(ctx context.Context, winnerId int) (duel.Duel, error) {
	return uc.duelService.ComposeDuel(ctx, winnerId)
}
