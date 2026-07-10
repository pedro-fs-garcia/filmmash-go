package admin

import (
	"context"
	"filmmash/internal/auth"
	"filmmash/internal/zitadel"
)

type AuthzFetcher interface {
	FetchAuthorizations(ctx context.Context, subs []string) (map[string]*zitadel.UserAuthz, error)
}

type Service struct {
	repo    *repository
	idp     *auth.Provider
	fetcher AuthzFetcher
}

func NewService(repo *repository, provider *auth.Provider, fetcher AuthzFetcher) *Service {
	return &Service{
		repo:    repo,
		idp:     provider,
		fetcher: fetcher,
	}
}

func (s *Service) GetUsersPaginated(ctx context.Context, pars PaginationParameters) (PaginatedUsers, error) {
	users, err := s.repo.GetUsersPaginated(ctx, pars)
	if err != nil {
		return PaginatedUsers{}, err
	}

	authz := map[string]*zitadel.UserAuthz{}
	if s.idp != nil && len(users) > 0 {
		subs := make([]string, 0, len(users))
		for _, u := range users {
			subs = append(subs, u.PidSub)
		}
		authz, err = s.fetcher.FetchAuthorizations(ctx, subs)
		if err != nil {
			return PaginatedUsers{}, err
		}
	}

	rows := make([]UserDashData, 0, len(users))
	for _, u := range users {
		rows = append(rows, UserDashData{
			User:  u,
			Authz: authz[u.PidSub],
		})
	}
	return toPaginatedUsers(pars.Size, rows), nil
}
