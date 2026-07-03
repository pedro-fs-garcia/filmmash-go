package admin

import (
	"filmmash/internal/auth"
	"time"

	"github.com/google/uuid"
)

type PaginationParameters struct {
	Size       int
	LastSeenId string
}

type UserWithVote struct {
	Id         uuid.UUID
	ZitadelSub string
	CreatedAt  time.Time
	Votes      int64
}

type UserDashData struct {
	User  UserWithVote
	Authz *auth.UserAuthz
}

type PaginatedUsers struct {
	Users []UserDashData
	Next  PaginationParameters
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
