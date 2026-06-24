package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"filmmash/internal/middleware"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type Handler struct {
	logger   *slog.Logger
	service  *Service
	provider *Zitadel
}

func NewHandler(logger *slog.Logger, provider *Zitadel, authService *Service) *Handler {
	return &Handler{
		logger:   logger,
		service:  authService,
		provider: provider,
	}
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "LoginHandler"),
		slog.String("request_id", reqId),
	)

	state := randString()
	verifier := randString()
	sum := sha256.Sum256([]byte(verifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(sum[:])

	authorizeUrl, err := h.provider.AuthorizeURL(state, codeChallenge)
	if err != nil {
		log.ErrorContext(ctx,
			"Failed to assemble auth provider authorize URL",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, NewOIDCFlowCookie(state, verifier))
	http.Redirect(w, r, authorizeUrl.String(), http.StatusFound)
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "LogoutHandler"),
		slog.String("request_id", reqId),
	)

	cookie, err := r.Cookie("session")
	if err != nil {
		log.DebugContext(ctx, "Failed to extract session cookie", slog.String("error", err.Error()))
		http.Error(w, "Failed to verify session cookie", http.StatusBadRequest)
		return
	}
	token := cookie.Value
	tokenHash := Sha256Hash(token)

	idToken, err := h.service.EndSession(ctx, tokenHash)
	if err != nil {
		log.ErrorContext(ctx, "Failed to end session", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.DebugContext(ctx, "check if idToken exists", slog.String("id_token", idToken))

	u, err := h.provider.EndSessionURL(idToken)
	if err != nil {
		log.ErrorContext(ctx,
			"Failed to parse auth provider base url into URL",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "CallbackHandler"),
		slog.String("request_id", reqId),
	)

	code := r.FormValue("code")
	state := r.FormValue("state")

	cookie, err := r.Cookie("oidc_flow")
	if err != nil {
		if err == http.ErrNoCookie {
			log.WarnContext(ctx,
				"No iodc_flow cookie found in request",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to verify authentication cookie", http.StatusUnauthorized)
			return
		}
		log.WarnContext(ctx,
			"Failed to read iodc_flow cookie in request",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	cookieState, cookieVerifier, ok := strings.Cut(cookie.Value, ":")
	if !ok {
		log.ErrorContext(ctx,
			"malformed cookie: could not parse iodc_flow cookie value",
			slog.String("error", "malformed iodc_flow cookie"),
		)
		http.Error(w, "failed to parse cookie value", http.StatusInternalServerError)
		return
	}

	if state != cookieState {
		log.DebugContext(ctx,
			fmt.Sprintf("expected state %s from iodc_flow cookie. Got %s.", state, cookieState),
			slog.String("error", "state extracted from cookie does not correpond to state received by auth provider"),
		)
		http.Error(w, "failed to parse cookie value", http.StatusInternalServerError)
		return
	}

	tokenResp, err := h.provider.RequestToken(code, cookieVerifier)
	if err != nil {
		log.ErrorContext(ctx,
			"Failed when requesting token from auth provider",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Failed to fetch tokens from auth provider", http.StatusBadRequest)
		return
	}

	session, rawToken, err := h.service.InitSession(ctx, tokenResp)
	if err != nil {
		log.ErrorContext(ctx, "Failed to init session", slog.String("error", err.Error()))
		http.Error(w, "failed to init session", http.StatusInternalServerError)
	}

	http.SetCookie(w, NewSessionCookie(session, rawToken))
	http.SetCookie(w, DeleteOIDCFlowCookie())
	http.Redirect(w, r, "/ui", http.StatusFound)
}
