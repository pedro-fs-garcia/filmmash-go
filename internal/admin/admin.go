package admin

import (
	"filmmash/internal/tmdb"
	"filmmash/internal/zitadel"
	"time"

	"github.com/google/uuid"
)

type PaginationParameters struct {
	Size       int
	LastSeenId string
}

type UserWithVote struct {
	Id        uuid.UUID
	PidSub    string
	CreatedAt time.Time
	Votes     int64
}

type UserDashData struct {
	User  UserWithVote
	Authz *zitadel.UserAuthz
}

type PaginatedUsers struct {
	Users []UserDashData
	Next  PaginationParameters
}

type TmdbSearchView struct {
	Movies   []tmdb.Movie
	Search   string
	NextPage int
	HasMore  bool
}

type TmdbAddResultView struct {
	Movie  tmdb.Movie
	Status string
	Added  bool
}

func toPaginatedUsers(size int, users []UserDashData) PaginatedUsers {
	resp := PaginatedUsers{Users: users}
	if len(users) > 0 {
		last := users[len(users)-1]
		resp.Next = PaginationParameters{
			Size:       size,
			LastSeenId: last.User.Id.String(),
		}
	}
	return resp
}
