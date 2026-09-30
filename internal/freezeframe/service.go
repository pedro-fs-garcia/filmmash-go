package freezeframe

import (
	"context"
	"filmmash/internal/database"
	"fmt"
	"log/slog"
	"time"
)

type Service struct {
	logger    *slog.Logger
	repo      *repository
	txManager *database.TxManager
	gameCache *GameCache
}

func NewService(logger *slog.Logger, repo *repository, txManager *database.TxManager) *Service {
	return &Service{
		logger:    logger,
		repo:      repo,
		txManager: txManager,
		gameCache: &GameCache{logger: logger},
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
		gameId, err = s.repo.InsertGame(txCtx, *g)
		if err != nil {
			return err
		}

		reelIds, err = s.repo.InsertReels(txCtx, gameId, g.Reels)
		if err != nil {
			return err
		}

		for i := range g.Reels {
			reel := &g.Reels[i]

			alternativeIds[i], err = s.repo.InsertReelAlternatives(txCtx, reelIds[i], reel.Alternatives)
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
			frameIds[i], err = s.repo.InsertFrames(txCtx, frames)
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
			reelFrameIds[i], err = s.repo.InsertReelFrames(txCtx, reelIds[i], dbReelFrames)
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

func (s *Service) GetGame(ctx context.Context, gameID int32) (Game, error) {
	if cached := s.gameCache.Read(); cached != nil && cached.ID == gameID {
		return *cached, nil
	}
	return s.repo.GetGame(ctx, gameID)
}

func (s *Service) GetTodaysGame(ctx context.Context) (Game, error) {
	cachedGame := s.gameCache.Read()

	year, month, day := time.Now().Date()
	if cachedGame != nil && cachedGame.ValidAt.Equal(time.Date(year, month, day, 0, 0, 0, 0, nil)) {
		return *s.gameCache.CachedGame, nil
	}

	gameIDs, err := s.repo.GetGameIDsByDate(ctx, time.Now())
	if len(gameIDs) == 0 {
		return Game{}, fmt.Errorf("%w: %w", ErrNoGameForDate, err)
	}

	game, err := s.GetGame(ctx, gameIDs[0])
	if err != nil {
		return Game{}, err
	}

	s.gameCache.Set(&game)
	return game, nil
}

func (s *Service) GetRound(ctx context.Context, seq int16) (Round, error) {
	g, err := s.GetTodaysGame(ctx)
	if err != nil {
		return Round{}, err
	}

	l := len(g.Reels)
	if seq <= 0 || int(seq) > l {
		return Round{}, fmt.Errorf("%w, seq must be between 1 and 5, got %d", ErrOutOfRange, seq)
	}

	r := g.Reels[seq-1]
	var nextReelID int32
	if int(seq) < l {
		nextReelID = g.Reels[seq].ID
	}

	return Round{
		GameID:       g.ID,
		ValidAt:      g.ValidAt,
		TotalReels:   l,
		ReelID:       r.ID,
		ReelSeq:      r.Seq,
		NextReelID:   nextReelID,
		ReelFrames:   r.ReelFrames,
		Alternatives: r.Alternatives,
	}, nil
}
