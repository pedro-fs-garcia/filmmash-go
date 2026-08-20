package main

import (
	"context"
	"filmmash/internal/admin"
	"filmmash/internal/auth"
	"filmmash/internal/config"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/metrics"
	"filmmash/internal/tmdb"
	"filmmash/internal/vote"
	"filmmash/internal/web"
	"filmmash/internal/zitadel"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func InitServer(port string, router *chi.Mux) (*http.Server, error) {
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", httpServer.Addr, err)
	}

	log.Printf("server listening on port %s\n", httpServer.Addr)

	if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return nil, err
	}
	return httpServer, nil
}

func initRouter(
	ctx context.Context,
	cfg *config.Config,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) (*chi.Mux, error) {
	txm := database.NewTxManager(pool)

	m := metrics.NewMetrics()

	tmdbClient := tmdb.NewClient("https://api.themoviedb.org/3", cfg.TmdbApitoken)

	filmRepo := film.NewRepository(pool)
	filmService := film.NewService(logger, filmRepo, txm, tmdbClient)
	filmHandler := film.NewHandler(logger, filmService)

	duelServiceCfg := duel.ServiceConfig{
		MinCandidates:        cfg.MinCandidates,
		MaxCandidates:        cfg.MaxCandidates,
		RatingWindow:         cfg.DuelRatingWindow,
		PopularityWeight:     cfg.DuelPopularityWeight,
		NewFilmBoost:         cfg.DuelNewFilmBoost,
		NewFilmDuelThreshold: cfg.DuelNewFilmDuelThreshold,
	}
	duelRepo := duel.NewRepository(pool)
	duelService := duel.NewService(m.DuelMetrics, duelRepo, txm, filmService, duelServiceCfg)
	duelHandler := duel.NewHandler(logger, duelService)

	voteRepo := vote.NewRepository(pool)
	regVoteUC := vote.NewRegisterVoteUC(m.VoteMetrics, voteRepo, txm, filmService, duelService)
	listVoteUC := vote.NewListVotesUC(voteRepo)
	voteHandler := vote.NewHandler(logger, regVoteUC, listVoteUC)

	zClient := zitadel.NewMachineTokenSource(
		logger,
		cfg.ZitadelBaseURL,
		cfg.ZitadelM2MClientId,
		cfg.ZitadelM2MClientSecret,
	)
	idp, err := auth.NewProvider(ctx,
		cfg.ZitadelBaseURL,
		cfg.ZitadelClientId,
		cfg.ZitadelClientSecret,
		cfg.AppBaseURL+"/auth/callback",
		cfg.AppBaseURL,
		&auth.ProviderConfig{RoleMapper: zitadel.RoleMapper},
	)
	if err != nil {
		return nil, err
	}

	oidcFlowCodec, err := auth.NewOIDCFlowCodec(cfg.AESKey)
	if err != nil {
		return nil, err
	}

	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(logger, idp, authRepo, txm)

	hcfg := auth.NewHandlerConfig("/ui", "/ui/films", cfg.ZitadelBaseURL+"/ui/console/users/me")
	authHandler := auth.NewHandler(logger, authService, idp, oidcFlowCodec, hcfg)

	adminRepo := admin.NewRepository(pool)
	adminService := admin.NewService(adminRepo, idp, zClient, tmdbClient, filmService)
	adminHandler := admin.NewHandler(logger, adminService, tmdbClient)

	router := web.NewRouter(
		logger,
		m.HttpMetrics,
		authService,
		authHandler,
		filmHandler,
		duelHandler,
		voteHandler,
		adminHandler,
	)

	router.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))

	go authService.DeleteExpiredSessionsJob(ctx, 24*time.Minute)

	go filmService.InsertFilmsJob(ctx, 24*time.Hour, tmdbClient.GetPopulars)
	go filmService.InsertFilmsJob(ctx, 24*7*time.Hour, tmdbClient.GetTopRated)

	return router, nil
}
