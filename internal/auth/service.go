package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"filmmash/internal/database"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	provider   *Zitadel
	repository *repository
	txManager  *database.TxManager
}

func NewService(
	logger *slog.Logger,
	provider *Zitadel,
	repo *repository,
) *Service {
	return &Service{
		logger:     logger.With("component", "Service"),
		provider:   provider,
		repository: repo,
		txManager:  database.NewTxManager(repo.pool),
	}
}

func (s *Service) InitSession(ctx context.Context, tokenResponse TokenResponse) (Session, SessionToken, string, error) {
	rawToken := randString()
	tokenHash := Sha256Hash(rawToken)

	idToken, err := s.provider.VerifyIdToken(ctx, tokenResponse.IDToken)
	if err != nil {
		return Session{}, "", "", err
	}

	var sessionDB SessionDB
	var displayName string

	err = s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		user := User{ZitadelSub: idToken.Subject}
		err := s.repository.UpsertUser(txCtx, &user)
		if err != nil {
			return err
		}

		userInfo, err := s.provider.GetUserInfo(ctx, tokenResponse.AccessToken)
		if err != nil {
			return err
		}
		displayName = userInfo.Name

		scopes := "openid profile email offline_access"
		accessTokenExpiresAt := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
		sessionDB = SessionDB{
			TokenHash:            tokenHash,
			UserID:               user.ID,
			AccessToken:          []byte(tokenResponse.AccessToken),
			AccessTokenExpiresAt: &accessTokenExpiresAt,
			IDToken:              []byte(tokenResponse.IDToken),
			Scopes:               &scopes,
			Roles:                userInfo.RoleKeys(),
			ExpiresAt:            time.Now().Add(time.Duration(360000) * time.Second),
		}

		err = s.repository.Insert(txCtx, &sessionDB)
		if err != nil {
			return err
		}

		sessionEvent := SessionEvent{
			SessionID:  sessionDB.ID,
			ZitadelSub: user.ZitadelSub,
			Event:      EventCreated,
			IPAddress:  sessionDB.IPAddress,
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
			SessionID:  session.ID,
			ZitadelSub: user.ZitadelSub,
			Event:      EventLoggedOut,
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
				log.ErrorContext(ctx, "deleting expired sessions")
			} else {
				log.Info(fmt.Sprintf("deleted %d expired sessions", total))
			}
		case <-ctx.Done():
			return
		}
	}
}

func NewReturntoCookie(url string) *http.Cookie {
	return &http.Cookie{
		Name:     "return_to",
		Value:    url,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   300,
		SameSite: http.SameSiteLaxMode,
	}
}

func DeleteReturnToCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "return_to",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // <0 writes "Max-Age: 0" → delete now
		HttpOnly: true,
		Secure:   true,
	}
}

func NewOIDCFlowCookie(state, verifier string) *OIDCFlowCookie {
	return &http.Cookie{
		Name:     "oidc_flow",
		Value:    state + ":" + verifier,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   300,
		SameSite: http.SameSiteLaxMode,
	}
}

func DeleteOIDCFlowCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "oidc_flow",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // <0 writes "Max-Age: 0" → delete now
		HttpOnly: true,
		Secure:   true,
	}
}

func NewSessionCookie(session Session, rawToken SessionToken) *SessionCookie {
	return &SessionCookie{
		Name:     "session",
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		SameSite: http.SameSiteLaxMode,
	}
}

func DeleteSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	}
}

func NewUserNameCookie(name string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     "user_name",
		Value:    strings.ReplaceAll(url.QueryEscape(name), "+", "%20"),
		Path:     "/",
		Secure:   true,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		SameSite: http.SameSiteLaxMode,
	}
}

func DeleteUserNameCookie() *http.Cookie {
	return &http.Cookie{
		Name:   "user_name",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
}
