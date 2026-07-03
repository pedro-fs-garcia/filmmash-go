package auth

import (
	"context"
	"net/http"
	"slices"
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

func (s *Service) SessionCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		session, err := s.repository.GetByTokenHash(r.Context(), Sha256Hash(cookie.Value))
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if time.Now().After(session.ExpiresAt) {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(userCtxKey).(Session)
	return session, ok
}

func RequiresAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := SessionFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !slices.Contains(session.Roles, "admin") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
