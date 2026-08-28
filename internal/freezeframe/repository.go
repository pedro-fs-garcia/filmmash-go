package freezeframe

import (
	"context"
	"filmmash/internal/database"
	"filmmash/internal/database/dbgen"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

func (r *repository) queries(ctx context.Context) *dbgen.Queries {
	return dbgen.New(database.ExtractTx(ctx, r.pool))
}

func (r *repository) insertGame(ctx context.Context, g *Game) error {
	id, err := r.queries(ctx).InsertGame(ctx, pgtype.Date{Time: g.ValidAt, Valid: true})
	if err != nil {
		return database.ParseDBError("inserting game", err)
	}

	g.ID = id
	return nil
}

func (r *repository) insertReelAlternatives(ctx context.Context, reelId int32, alts []Alternative) ([]int32, error) {
	if len(alts) == 0 {
		return nil, nil
	}

	params := make([]dbgen.InsertReelAlternativesParams, len(alts))
	for i, a := range alts {
		params[i] = dbgen.InsertReelAlternativesParams{
			ReelID: reelId,
			FilmID: int32(a.Film.Id),
			Seq:    a.Seq,
		}
	}

	res := r.queries(ctx).InsertReelAlternatives(ctx, params)
	defer func() { _ = res.Close() }()

	ids := make([]int32, len(alts))
	var batchErr error
	res.QueryRow(func(i int, id int32, err error) {
		if err != nil {
			if batchErr == nil {
				batchErr = database.ParseDBError(
					fmt.Sprintf(
						"inserting reel alternative (reel_id: %d, film_id: %d, seq: %d)",
						reelId, params[i].FilmID, params[i].Seq,
					),
					err,
				)
			}
			return
		}
		ids[i] = id
	})
	if batchErr != nil {
		return nil, batchErr
	}
	return ids, nil
}

// func (uc *SeedGameUC) Seed(ctx context.Context, g *Game) error {
// 	if err := g.Validate(); err != nil {
// 		return err
// 	}
// 	return uc.txManager.ExecTx(ctx, func(txCtx context.Context) error {
// 		if err := uc.repo.insertGame(txCtx, g); err != nil {
// 			return err
// 		}
// 		reelIDs, err := uc.repo.insertReels(txCtx, g.ID, g.Reels)
// 		if err != nil {
// 			return err
// 		}
// 		for i := range g.Reels {
// 			if _, err := uc.repo.insertReelAlternatives(txCtx, reelIDs[i], g.Reels[i].Alternatives); err != nil {
// 				return err
// 			}
// 			// frames, then reel_frames
// 		}
// 		return nil
// 	})
// }
