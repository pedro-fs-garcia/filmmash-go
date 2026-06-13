package vote

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type RegisterVoteUC struct {
	voteRepo    *repository
	txManager   *database.TxManager
	filmService *film.Service
	duelService *duel.Service
}

func NewRegisterVoteUC(repo *repository, filmService *film.Service, duelService *duel.Service) *RegisterVoteUC {
	return &RegisterVoteUC{
		voteRepo:    repo,
		txManager:   database.NewTxManager(repo.pool),
		filmService: filmService,
		duelService: duelService,
	}
}

func (uc *RegisterVoteUC) RegisterVote(ctx context.Context, duelId uuid.UUID, winnerId int) (Vote, error) {
	var vote Vote
	err := uc.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		ratings, err := uc.duelService.GetDuelRatings(txCtx, duelId)
		if err != nil {
			log.Println(err)
			return nil
		}

		var winner, loser film.FilmRating
		switch winnerId {
		case ratings[0].Id:
			winner, loser = ratings[0], ratings[1]
		case ratings[1].Id:
			winner, loser = ratings[1], ratings[0]
		default:
			err = fmt.Errorf("winner does not belong to duel %s", duelId.String())
			log.Println(err)
			return err
		}

		newWinnerRating, newLoserRating := film.CalculateRatings(winner.Rating, loser.Rating)
		vote = Vote{
			DuelId:            duelId,
			WinnerID:          winnerId,
			LoserId:           loser.Id,
			WinnerRatingAfter: newWinnerRating,
			LoserRatingAfter:  newLoserRating,
		}
		err = uc.voteRepo.InsertVote(txCtx, &vote)
		if err != nil {
			log.Println(err)
			return err
		}

		filmRatings := []*film.FilmRating{
			{Id: winnerId, Rating: newWinnerRating},
			{Id: loser.Id, Rating: newLoserRating},
		}
		err = uc.filmService.UpdateRatings(txCtx, filmRatings)
		return err
	})
	if err != nil {
		return Vote{}, err
	}
	return vote, nil
}
