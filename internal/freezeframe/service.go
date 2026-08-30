package freezeframe

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"log/slog"
)

type Service struct {
	logger    *slog.Logger
	repo      *repository
	txManager *database.TxManager
}

func NewService(logger *slog.Logger, repo *repository, txManager *database.TxManager) *Service {
	return &Service{
		logger:    logger,
		repo:      repo,
		txManager: txManager,
	}
}

func (s *Service) SeedGame(ctx context.Context, g *Game) error {
	var seedErr error
	if seedErr = g.Validate(); seedErr != nil {
		return seedErr
	}

	var gameId int32
	var reelIds []int32
	var alternativeIds = make([][]int32, len(g.Reels))
	var reelFrameIds = make([][]int32, len(g.Reels))

	seedErr = s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		gameId, err = s.repo.insertGame(txCtx, g)
		if err != nil {
			return err
		}

		reelIds, err = s.repo.insertReels(txCtx, gameId, g.Reels)
		if err != nil {
			return err
		}

		for i := range g.Reels {
			alternativeIds[i], err = s.repo.insertReelAlternatives(txCtx, reelIds[i], g.Reels[i].Alternatives)
			if err != nil {
				return err
			}

			frames := make([]Frame, len(g.Reels[i].ReelFrames))
			for j, fr := range g.Reels[i].ReelFrames {
				frames[j] = Frame{
					FilmID:    int32(g.Reels[i].Film.Id),
					ImagePath: fr.ImagePath,
				}
			}
			frameIds, err := s.repo.insertFrames(txCtx, frames)
			if err != nil {
				return err
			}

			dbReelFrames := make([]dbgen.ReelFrame, len(g.Reels[i].ReelFrames))
			for k, rf := range g.Reels[i].ReelFrames {
				dbReelFrames[k] = dbgen.ReelFrame{
					ReelID:     reelIds[i],
					FrameID:    frameIds[k],
					Difficulty: rf.Difficulty,
					Seq:        rf.Seq,
				}
			}
			reelFrameIds[i], err = s.repo.insertReelFrames(txCtx, dbReelFrames)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if seedErr != nil {
		return seedErr
	}

	g.ID = gameId
	for i := range g.Reels {
		g.Reels[i].ID = reelIds[i]
		for j := range g.Reels[i].ReelFrames {
			g.Reels[i].ReelFrames[j].ID = reelFrameIds[i][j]
		}
		for k := range g.Reels[i].Alternatives {
			g.Reels[i].Alternatives[k].ID = alternativeIds[i][k]
		}
	}

	return nil
}
