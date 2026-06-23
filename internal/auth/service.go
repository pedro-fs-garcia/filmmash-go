package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"filmmash/internal/database"
	"net/http"
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
	provider         *ZitadelProvider
	sessionRepo      *SessionRepository
	sessionEventRepo *SessionEventRepository
	txManager        *database.TxManager
}

func NewService(
	provider *ZitadelProvider,
	repo *SessionRepository,
	sessionEventRepo *SessionEventRepository,
) *Service {
	return &Service{
		provider:         provider,
		sessionRepo:      repo,
		sessionEventRepo: sessionEventRepo,
		txManager:        database.NewTxManager(repo.pool),
	}
}

func (s *Service) InitSession(ctx context.Context, tokenResponse TokenResponse) (Session, SessionToken, error) {
	rawToken := randString()
	tokenHash := Sha256Hash(rawToken)

	idToken, err := s.provider.VerifyIdToken(ctx, tokenResponse.IDToken)
	if err != nil {
		return Session{}, "", err
	}

	scopes := "openid profile email offline_access"
	accessTokenExpiresAt := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	sessionDB := SessionDB{
		TokenHash:            tokenHash,
		UserID:               idToken.Subject,
		AccessToken:          []byte(tokenResponse.AccessToken),
		AccessTokenExpiresAt: &accessTokenExpiresAt,
		IDToken:              []byte(tokenResponse.IDToken),
		Scopes:               &scopes,
		ExpiresAt:            time.Now().Add(time.Duration(360000) * time.Second),
	}

	err = s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		err = s.sessionRepo.Insert(ctx, &sessionDB)
		if err != nil {
			return err
		}

		sessionEvent := SessionEvent{
			SessionID: sessionDB.ID,
			UserID:    sessionDB.UserID,
			Event:     EventCreated,
			IPAddress: sessionDB.IPAddress,
		}
		err = s.sessionEventRepo.Insert(ctx, &sessionEvent)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return Session{}, "", err
	}

	return SessionDBToSession(sessionDB), rawToken, nil

	// var claims struct {
	// 	Email string `json:"email"`
	// 	Name  string `json:"name"`
	// }
	// idToken.Claims(&claims)

	// opaqueId := randString()
	// sessionLifetime := 300
}

func (s *Service) EndSession(ctx context.Context, tokenHash []byte) (string, error) {
	session, err := s.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", err
	}
	err = s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		sessionEvent := SessionEvent{
			SessionID: session.ID,
			UserID:    session.UserID,
			Event:     EventCreated,
		}
		err = s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		if err != nil {
			return err
		}
		return s.sessionEventRepo.Insert(ctx, &sessionEvent)
	})
	if err != nil {
		return "", err
	}
	return session.IDToken, nil
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
