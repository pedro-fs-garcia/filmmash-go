package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type Handler struct {
	logger   *slog.Logger
	service  *Service
	provider *ZitadelProvider
}

func NewHandler(logger *slog.Logger, provider *ZitadelProvider, authService *Service) *Handler {
	return &Handler{
		logger:   logger,
		service:  authService,
		provider: provider,
	}
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state := randString()
	verifier := randString()
	sum := sha256.Sum256([]byte(verifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(sum[:])

	u, _ := url.Parse(fmt.Sprintf("%s/oauth/v2/authorize", h.provider.providerUrl))
	q := u.Query()
	q.Set("client_id", h.provider.clientId)
	q.Set("redirect_uri", "http://localhost:8080/auth/callback")
	q.Set("response_type", "code")
	q.Set("scope", "openid profile email offline_access")
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	redirect := u.String()

	http.SetCookie(w, NewOIDCFlowCookie(state, verifier))
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Failed to verify session cookie", http.StatusBadRequest)
	}
	token := cookie.Value
	tokenHash := Sha256Hash(token)

	idToken, err := h.service.EndSession(r.Context(), tokenHash)

	u, _ := url.Parse(fmt.Sprintf("%s/oidc/v1/end_session", h.provider.providerUrl))
	q := u.Query()
	q.Set("id_token_hint", idToken)
	q.Set("post_logout_redirect_uri", "http://localhost:8080/ui/films")
	u.RawQuery = q.Encode()
	redirect := u.String()
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (h *Handler) RequestToken(code, verifier string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:8080/auth/callback")
	form.Set("code_verifier", verifier)

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/oauth/v2/token", h.provider.providerUrl),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return TokenResponse{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.SetBasicAuth(h.provider.clientId, h.provider.clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TokenResponse{}, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return TokenResponse{}, err
	}

	return tokenResp, nil
}

func (h *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.logger.With(
		slog.String("method", "CallbackHandler"),
	)

	code := r.FormValue("code")
	state := r.FormValue("state")

	cookie, err := r.Cookie("oidc_flow")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Failed to verify authentication cookie", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	parsed := strings.Split(cookie.Value, ":")
	cookieState := parsed[0]
	cookieVerifier := parsed[1]

	if state != cookieState {
		http.Error(w, "failed to parse cookie value", http.StatusInternalServerError)
	}

	tokenResp, err := h.RequestToken(code, cookieVerifier)
	if err != nil {
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
