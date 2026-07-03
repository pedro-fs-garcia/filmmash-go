package admin

import (
	"context"
	"filmmash/internal/auth"
)

type Service struct {
	repo    *repository
	zitadel *auth.Zitadel
}

func NewService(repo *repository, provider *auth.Zitadel) *Service {
	return &Service{
		repo:    repo,
		zitadel: provider,
	}
}

func (s *Service) GetUsersPaginated(ctx context.Context, pars PaginationParameters) (PaginatedUsers, error) {
	users, err := s.repo.GetUsersPaginated(ctx, pars)
	if err != nil {
		return PaginatedUsers{}, err
	}

	authz := map[string]*auth.UserAuthz{}
	if s.zitadel != nil && len(users) > 0 {
		subs := make([]string, 0, len(users))
		for _, u := range users {
			subs = append(subs, u.ZitadelSub)
		}
		authz, err = s.zitadel.FetchAuthorizations(ctx, subs)
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
