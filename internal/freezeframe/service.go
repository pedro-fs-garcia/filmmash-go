package freezeframe

import (
	"context"
	"filmmash/internal/database"
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
	if err := g.Validate(); err != nil {
		return err
	}

	var seedErr error
	var gameId int32
	var reelIds []int32
	var alternativeIds = make([][]int32, len(g.Reels))
	var frameIds = make([][]int32, len(g.Reels))
	var reelFrameIds = make([][]int32, len(g.Reels))

	seedErr = s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		gameId, err = s.repo.insertGame(txCtx, *g)
		if err != nil {
			return err
		}

		reelIds, err = s.repo.insertReels(txCtx, gameId, g.Reels)
		if err != nil {
			return err
		}

		for i := range g.Reels {
			reel := &g.Reels[i]

			alternativeIds[i], err = s.repo.insertReelAlternatives(txCtx, reelIds[i], reel.Alternatives)
			if err != nil {
				return err
			}

			frames := make([]Frame, len(reel.ReelFrames))
			for j, fr := range reel.ReelFrames {
				frames[j] = Frame{
					FilmID:    fr.Frame.FilmID,
					ImagePath: fr.Frame.ImagePath,
				}
			}
			frameIds[i], err = s.repo.insertFrames(txCtx, frames)
			if err != nil {
				return err
			}

			dbReelFrames := make([]ReelFrame, len(g.Reels[i].ReelFrames))
			for k, rf := range g.Reels[i].ReelFrames {
				dbReelFrames[k] = ReelFrame{
					Difficulty: rf.Difficulty,
					Seq:        rf.Seq,
					Frame:      Frame{ID: frameIds[i][k]},
				}
			}
			reelFrameIds[i], err = s.repo.insertReelFrames(txCtx, reelIds[i], dbReelFrames)
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
			g.Reels[i].ReelFrames[j].Frame.ID = frameIds[i][j]
		}
		for k := range g.Reels[i].Alternatives {
			g.Reels[i].Alternatives[k].ID = alternativeIds[i][k]
		}
	}

	return nil
}
