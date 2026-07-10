package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"filmmash/internal/database"
	"fmt"
	"log/slog"
	"net/netip"
	"time"
)

const maxUserAgentLen = 512

func randString() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func Sha256Hash(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

type Service struct {
	logger     *slog.Logger
	idp        *Provider
	repository *repository
	txManager  *database.TxManager
}

func NewService(
	logger *slog.Logger,
	idp *Provider,
	repo *repository,
	txm *database.TxManager,
) *Service {
	return &Service{
		logger:     logger.With("component", "Service"),
		idp:        idp,
		repository: repo,
		txManager:  txm,
	}
}

func (s *Service) InitSession(
	ctx context.Context,
	tokens *AuthTokens,
	ip netip.Addr,
	userAgent string,
) (Session, SessionToken, string, error) {
	rawToken := randString()
	tokenHash := Sha256Hash(rawToken)

	if len(userAgent) > maxUserAgentLen {
		userAgent = userAgent[:maxUserAgentLen]
	}
	var userAgentPtr *string
	if userAgent != "" {
		userAgentPtr = &userAgent
	}

	var sessionDB SessionDB
	var displayName string

	err := s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		user := User{PidSub: tokens.IDToken.Subject}
		err := s.repository.UpsertUser(txCtx, &user)
		if err != nil {
			return err
		}

		userInfo, err := s.idp.GetUserInfo(ctx, tokens.Token.AccessToken)
		if err != nil {
			return err
		}
		displayName = userInfo.Name

		scopes := s.idp.OauthScopes()
		sessionDB = SessionDB{
			TokenHash:            tokenHash,
			UserID:               user.ID,
			AccessToken:          []byte(tokens.Token.AccessToken),
			AccessTokenExpiresAt: &tokens.Token.Expiry,
			RefreshToken:         []byte(tokens.Token.RefreshToken),
			IDToken:              []byte(tokens.RawIDToken),
			Scopes:               &scopes,
			IPAddress:            ip,
			UserAgent:            userAgentPtr,
			Roles:                userInfo.Roles,
			ExpiresAt:            time.Now().Add(time.Duration(360000) * time.Second),
		}

		err = s.repository.Insert(txCtx, &sessionDB)
		if err != nil {
			return err
		}

		sessionEvent := SessionEvent{
			SessionID: sessionDB.ID,
			PidSub:    user.PidSub,
			Event:     EventCreated,
			IPAddress: sessionDB.IPAddress,
		}
		err = s.repository.InsertEvent(txCtx, &sessionEvent)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return Session{}, "", "", err
	}

	return SessionDBToSession(sessionDB), rawToken, displayName, nil
}

func (s *Service) EndSession(ctx context.Context, tokenHash []byte) (string, error) {
	session, err := s.repository.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", err
	}
	err = s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		user, err := s.repository.GetUserById(ctx, session.UserID)
		if err != nil {
			return err
		}

		sessionEvent := SessionEvent{
			SessionID: session.ID,
			PidSub:    user.PidSub,
			Event:     EventLoggedOut,
		}
		err = s.repository.DeleteByTokenHash(ctx, tokenHash)
		if err != nil {
			return err
		}
		return s.repository.InsertEvent(ctx, &sessionEvent)
	})
	if err != nil {
		return "", err
	}
	return session.IDToken, nil
}

func (s *Service) DeleteExpiredSessionsJob(ctx context.Context, duration time.Duration) {
	log := s.logger.With("method", "DeleteExpiredSessionsJob")
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			total, err := s.repository.DeleteExpiredSessions(ctx)
			if err != nil {
				log.ErrorContext(ctx, "deleting expired sessions", "error", err)
			} else {
				log.Info(fmt.Sprintf("deleted %d expired sessions", total))
			}
		case <-ctx.Done():
			return
		}
	}
}
