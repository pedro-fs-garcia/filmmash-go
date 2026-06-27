package auth

import (
	"context"
	"net/http"
	"time"
)

type ctxKey struct{ name string }

var userCtxKey = ctxKey{"authUser"}

func (s *Service) SessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			next(w, r)
			return
		}
		tokenHash := Sha256Hash(cookie.Value)
		session, err := s.repository.GetByTokenHash(r.Context(), tokenHash)
		if err != nil {
			next(w, r)
			return
		}
		if time.Now().After(session.ExpiresAt) {
			next(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey, session)
		next(w, r.WithContext(ctx))
	}
}

func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(userCtxKey).(Session)
	return session, ok
}
