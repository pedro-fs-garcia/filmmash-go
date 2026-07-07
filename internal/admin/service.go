package admin

import (
	"context"
	"filmmash/internal/auth"
)

type Service struct {
	repo *repository
	idp  *auth.Provider
}

func NewService(repo *repository, provider *auth.Provider) *Service {
	return &Service{
		repo: repo,
		idp:  provider,
	}
}

func (s *Service) GetUsersPaginated(ctx context.Context, pars PaginationParameters) (PaginatedUsers, error) {
	users, err := s.repo.GetUsersPaginated(ctx, pars)
	if err != nil {
		return PaginatedUsers{}, err
	}

	authz := map[string]*auth.UserAuthz{}
	if s.idp != nil && len(users) > 0 {
		subs := make([]string, 0, len(users))
		for _, u := range users {
			subs = append(subs, u.ZitadelSub)
		}
		authz, err = s.idp.FetchAuthorizations(ctx, subs)
		if err != nil {
			return PaginatedUsers{}, err
		}
	}

	rows := make([]UserDashData, 0, len(users))
	for _, u := range users {
		rows = append(rows, UserDashData{
			User:  u,
			Authz: authz[u.ZitadelSub],
		})
	}
	return toPaginatedUsers(pars.Size, rows), nil
}
