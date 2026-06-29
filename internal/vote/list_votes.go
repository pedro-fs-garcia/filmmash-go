package vote

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VotedFilm struct {
	Id          int
	Title       string
	RatingAfter float64
	Year        int
	ImagePath   string
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

func (uc *ListVotesUC) ListMyVotes(ctx context.Context, userId uuid.UUID) ([]MatchupResult, error) {
	return uc.voteRepo.ListUsersVotes(ctx, userId)
}
