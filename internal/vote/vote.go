package vote

import "github.com/google/uuid"

type Vote struct {
	Id                uuid.UUID `db:"id"`
	DuelId            uuid.UUID `db:"duel_id"`
	WinnerID          int       `db:"winner_id"`
	LoserId           int       `db:"loser_id"`
	WinnerRatingAfter float64   `db:"winner_rating_after"`
	LoserRatingAfter  float64   `db:"loser_rating_after"`
	AppliedSeq        int64     `db:"applied_seq"`
}
