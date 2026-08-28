package freezeframe

import (
	"filmmash/internal/database"
	"log/slog"
)

type Service struct {
	logger    *slog.Logger
	repo      *repository
	txManager *database.TxManager
}

func NewService(logger *slog.Logger, repo *repository, txManager *database.TxManager) *Service {
	return &Service{
		logger:    logger,
		repo:      repo,
		txManager: txManager,
	}
}
