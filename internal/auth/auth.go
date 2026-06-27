package auth

import (
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SessionCookie = http.Cookie
type OIDCFlowCookie = http.Cookie
type SessionToken = string

type SessionEventType string

const (
	EventCreated   SessionEventType = "created"
	EventLoggedOut SessionEventType = "logged_out"
	EventExpired   SessionEventType = "expired"
	EventRevoked   SessionEventType = "revoked"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

type User struct {
	ID         uuid.UUID
	ZitadelSub string
	CreatedAt  time.Time
}

type SessionDB struct {
	ID                   uuid.UUID  `db:"id"`
	TokenHash            []byte     `db:"token_hash"`
	UserID               uuid.UUID  `db:"user_id"`
	AccessToken          []byte     `db:"access_token"`
	AccessTokenExpiresAt *time.Time `db:"access_token_expires_at"`
	RefreshToken         []byte     `db:"refresh_token"`
	IDToken              []byte     `db:"id_token"`
	Scopes               *string    `db:"scopes"`
	IPAddress            netip.Addr `db:"ip_address"`
	UserAgent            *string    `db:"user_agent"`
	CreatedAt            time.Time  `db:"created_at"`
	LastSeenAt           time.Time  `db:"last_seen_at"`
	ExpiresAt            time.Time  `db:"expires_at"`
}

type SessionEvent struct {
	ID         int64            `db:"id"`
	SessionID  uuid.UUID        `db:"session_id"`
	ZitadelSub string           `db:"zitadel_sub"`
	Event      SessionEventType `db:"event"`
	IPAddress  netip.Addr       `db:"ip_address"`
	CreatedAt  time.Time        `db:"created_at"`
}

type Session struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	AccessToken          string
	AccessTokenExpiresAt time.Time
	RefreshToken         string
	IDToken              string
	Scopes               []string
	IPAddress            netip.Addr
	UserAgent            string
	CreatedAt            time.Time
	LastSeenAt           time.Time
	ExpiresAt            time.Time
}

type UserInfo struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	EmailVerified     bool   `json:"email_verified"`
	FamilyName        string `json:"family_name"`
	GivenName         string `json:"given_name"`
	Locale            string `json:"locale"`
	PreferredUsername string `json:"preferred_username"`
	UpdatedAt         int64  `json:"updated_at"`
}

func SessionToSessionDB(session Session, tokenHash []byte) SessionDB {
	scope := strings.Join(session.Scopes, " ")
	sessionDB := SessionDB{
		ID:                   session.ID,
		TokenHash:            tokenHash,
		UserID:               session.UserID,
		AccessToken:          []byte(session.AccessToken),
		AccessTokenExpiresAt: &session.AccessTokenExpiresAt,
		RefreshToken:         []byte(session.RefreshToken),
		IDToken:              []byte(session.IDToken),
		Scopes:               &scope,
		IPAddress:            session.IPAddress,
		UserAgent:            &session.UserAgent,
		CreatedAt:            session.CreatedAt,
		LastSeenAt:           session.LastSeenAt,
		ExpiresAt:            session.ExpiresAt,
	}
	if len(scope) == 0 {
		sessionDB.Scopes = nil
	}
	return sessionDB
}

func SessionDBToSession(sdb SessionDB) Session {
	session := Session{
		ID:           sdb.ID,
		UserID:       sdb.UserID,
		AccessToken:  string(sdb.AccessToken),
		RefreshToken: string(sdb.RefreshToken),
		IDToken:      string(sdb.IDToken),
		IPAddress:    sdb.IPAddress,
		CreatedAt:    sdb.CreatedAt,
		LastSeenAt:   sdb.LastSeenAt,
		ExpiresAt:    sdb.ExpiresAt,
	}
	if sdb.AccessTokenExpiresAt != nil {
		session.AccessTokenExpiresAt = *sdb.AccessTokenExpiresAt
	}
	if sdb.Scopes != nil {
		session.Scopes = strings.Split(*sdb.Scopes, " ")
	}
	if sdb.UserAgent != nil {
		session.UserAgent = *sdb.UserAgent
	}
	return session
}
