package admin

import (
	"context"
	"filmmash/internal/auth"
	"filmmash/internal/film"
	"filmmash/internal/tmdb"
	"filmmash/internal/zitadel"
)

type AuthzFetcher interface {
	FetchAuthorizations(ctx context.Context, subs []string) (map[string]*zitadel.UserAuthz, error)
}

type Service struct {
	repo        *repository
	idp         *auth.Provider
	fetcher     AuthzFetcher
	tmdbClient  *tmdb.Client
	filmService *film.Service
}

func NewService(
	repo *repository, provider *auth.Provider, fetcher AuthzFetcher, tmdbClient *tmdb.Client, filmService *film.Service,
) *Service {
	return &Service{
		repo:        repo,
		idp:         provider,
		fetcher:     fetcher,
		tmdbClient:  tmdbClient,
		filmService: filmService,
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

func (s *Service) InsertFilm(ctx context.Context, film *film.Film) error {
	return s.filmService.InsertFilm(ctx, film)
}

func (s *Service) MarkInCatalogue(ctx context.Context, movies []tmdb.Movie) error {
	ids := make([]int32, len(movies))
	for i, m := range movies {
		ids[i] = int32(m.Id)
	}

	in, err := s.filmService.IdsInCatalogue(ctx, ids)
	if err != nil {
		return err
	}

	for i := range movies {
		movies[i].InCatalogue = in[int32(movies[i].Id)]
	}
	return nil
}
