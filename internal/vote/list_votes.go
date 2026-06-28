package vote

import (
	"context"
	"time"
)

type VotedFilm struct {
	Id          int
	Title       string
	RatingAfter float64
	Year        int
}

type MatchupResult struct {
	Winner    VotedFilm
	Loser     VotedFilm
	CreatedAt time.Time
}

type ListVotesUC struct {
	voteRepo *repository
}

func NewListVotesUC(repo *repository) *ListVotesUC {
	return &ListVotesUC{
		voteRepo: repo,
	}
}

func (uc *ListVotesUC) ListVotes(ctx context.Context, filmId int) ([]MatchupResult, error) {
	return uc.voteRepo.ListFilmVotes(ctx, filmId)
}
