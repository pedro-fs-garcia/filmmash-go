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

func (r *repository) insertGame(ctx context.Context, g Game) (int32, error) {
	id, err := r.queries(ctx).InsertGame(ctx, pgtype.Date{Time: g.ValidAt, Valid: true})
	if err != nil {
		return 0, database.ParseDBError("inserting game", err)
	}
	return id, nil
}

func (r *repository) insertReels(ctx context.Context, gameId int32, reels []Reel) ([]int32, error) {
	if len(reels) == 0 {
		return nil, nil
	}

	params := make([]dbgen.InsertReelsParams, len(reels))
	for i, reel := range reels {
		params[i] = dbgen.InsertReelsParams{
			GameID: gameId,
			FilmID: int32(reel.Film.Id),
			Seq:    reel.Seq,
		}
	}

	res := r.queries(ctx).InsertReels(ctx, params)
	defer func() { _ = res.Close() }()

	ids := make([]int32, len(reels))
	var batchErr error
	res.QueryRow(func(i int, id int32, err error) {
		if err != nil {
			if batchErr == nil {
				batchErr = database.ParseDBError(
					fmt.Sprintf(
						"inserting reel (game_id: %d, film_id: %d, seq: %d)",
						gameId, params[i].FilmID, params[i].Seq,
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

func (r *repository) insertFrames(ctx context.Context, frames []Frame) ([]int32, error) {
	if len(frames) == 0 {
		return nil, nil
	}

	params := make([]dbgen.InsertFramesParams, len(frames))
	for i, f := range frames {
		params[i] = dbgen.InsertFramesParams{
			FilmID:    f.FilmID,
			ImagePath: f.ImagePath,
		}
	}

	res := r.queries(ctx).InsertFrames(ctx, params)
	defer func() { _ = res.Close() }()

	ids := make([]int32, len(frames))
	var batchErr error
	res.QueryRow(func(i int, id int32, err error) {
		if err != nil {
			if batchErr == nil {
				batchErr = database.ParseDBError(
					fmt.Sprintf(
						"inserting frame (film_id: %d, image_path: %s)",
						frames[i].FilmID, frames[i].ImagePath,
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

func (r *repository) insertReelFrames(ctx context.Context, reelId int32, reelFrames []ReelFrame) ([]int32, error) {
	if len(reelFrames) == 0 {
		return nil, nil
	}

	params := make([]dbgen.InsertReelFramesParams, len(reelFrames))
	for i, rf := range reelFrames {
		params[i] = dbgen.InsertReelFramesParams{
			ReelID:     reelId,
			FrameID:    rf.Frame.ID,
			Difficulty: rf.Difficulty,
			Seq:        rf.Seq,
		}
	}

	res := r.queries(ctx).InsertReelFrames(ctx, params)
	defer func() { _ = res.Close() }()

	ids := make([]int32, len(reelFrames))
	var batchErr error

	res.QueryRow(func(i int, id int32, err error) {
		if err != nil {
			if batchErr == nil {
				rf := reelFrames[i]
				batchErr = database.ParseDBError(
					fmt.Sprintf(
						"inserting reel frame (reel_id: %d, frame_id: %d, difficulty: %d, seq: %d)",
						reelId, rf.Frame.ID, rf.Difficulty, rf.Seq,
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
