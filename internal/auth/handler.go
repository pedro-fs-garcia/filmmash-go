package auth

import (
	"filmmash/internal/middleware"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type HandlerConfig struct {
	defaultReturnToPath string
	postLogoutPath      string
	accountConsoleURL   string
}

func NewHandlerConfig(
	defaultReturntoPath, postLogoutPath, accountConsoleURL string,
) *HandlerConfig {
	return &HandlerConfig{
		defaultReturnToPath: defaultReturntoPath,
		postLogoutPath:      postLogoutPath,
		accountConsoleURL:   accountConsoleURL,
	}
}

type Handler struct {
	logger        *slog.Logger
	service       *Service
	idp           *Provider
	oidcFlowCodec *OIDCFlowCodec
	cfg           *HandlerConfig
}

func NewHandler(
	logger *slog.Logger,
	authService *Service,
	idp *Provider,
	oidcFlowCodec *OIDCFlowCodec,
	cfg *HandlerConfig,
) *Handler {
	return &Handler{
		logger:        logger,
		service:       authService,
		idp:           idp,
		oidcFlowCodec: oidcFlowCodec,
		cfg:           cfg,
	}
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "LoginHandler"),
		slog.String("request_id", reqId),
	)

	returnTo := r.URL.Query().Get("next")
	if returnTo == "" {
		returnTo = "/"
	}
	if !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		log.WarnContext(ctx, "invalid next parameter", slog.String("next", returnTo))
		http.Error(w, "invalid next parameter", http.StatusBadRequest)
		return
	}

	lc, err := h.idp.NewLoginChallenge()
	if err != nil {
		log.ErrorContext(ctx,
			"Failed to assemble auth provider authorize URL",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.oidcFlowCodec.SetCookie(w, lc.State, lc.Nonce, lc.Verifier)
	if err != nil {
		log.ErrorContext(ctx,
			"Failed to set auth flow cookie",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	SetReturnToCookie(w, returnTo)
	http.Redirect(w, r, lc.AuthorizeURL, http.StatusFound)
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
		http.Redirect(w, r, h.cfg.postLogoutPath, http.StatusFound)
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

	u, err := h.idp.EndSessionURL(idToken)
	if err != nil {
		log.ErrorContext(ctx,
			"Failed to parse auth provider base url into URL",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	DropSessionCookie(w)
	DropUserNameCookie(w)
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "CallbackHandler"),
		slog.String("request_id", reqId),
	)

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	cookieState, cookieNonce, cookieVerifier, err := h.oidcFlowCodec.ReadCookie(r)
	if err != nil {
		log.WarnContext(ctx,
			fmt.Sprintf("Failed to read %s cookie in request", cookieName),
			slog.String("error", err.Error()),
		)
		http.Error(w, "No auth cookie was found in request", http.StatusBadRequest)
		return
	}

	if state != cookieState {
		log.DebugContext(ctx,
			fmt.Sprintf("expected state %s from iodc_flow cookie. Got %s.", state, cookieState),
			slog.String("error", "state extracted from cookie does not correpond to state received by auth provider"),
		)
		http.Error(w, "state mismatch for login flow", http.StatusBadRequest)
		return
	}

	authTokens, err := h.idp.RequestToken(ctx, code, cookieVerifier, cookieNonce)
	if err != nil {
		log.ErrorContext(ctx,
			"Failed when requesting token from auth provider",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Failed to fetch tokens from auth provider", http.StatusBadRequest)
		return
	}

	ip := chimiddleware.GetClientIPAddr(ctx)
	session, rawToken, displayName, err := h.service.InitSession(ctx, authTokens, ip, r.UserAgent())
	if err != nil {
		log.ErrorContext(ctx, "Failed to init session", slog.String("error", err.Error()))
		http.Error(w, "failed to init session", http.StatusInternalServerError)
		return
	}

	var returnTo string
	returnToCookie, err := r.Cookie("return_to")
	if err != nil {
		returnTo = h.cfg.defaultReturnToPath
	} else {
		returnTo = returnToCookie.Value
	}

	SetSessionCookie(w, session, rawToken)
	SetUserNameCookie(w, displayName, session.AccessTokenExpiresAt)
	ClearOIDCFlowCookie(w)
	DropReturnToCookie(w)
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (h *Handler) UserConsoleHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.cfg.accountConsoleURL, http.StatusFound)
}
