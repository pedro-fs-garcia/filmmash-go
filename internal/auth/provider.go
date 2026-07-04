package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Provider struct {
	oauth     *oauth2.Config
	verifier  *oidc.IDTokenVerifier
	logoutURL string
}

func NewProvider(ctx context.Context,
	issuer, clientId, clientSecret, callbackURL string,
) (*Provider, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	var meta struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&meta); err != nil {
		return nil, err
	}

	return &Provider{
		oauth: &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email", oidc.ScopeOfflineAccess},
		},
		verifier:  provider.Verifier(&oidc.Config{ClientID: clientId}),
		logoutURL: meta.EndSessionEndpoint,
	}, nil
}

func (p *Provider) Login(w http.ResponseWriter, r *http.Request) {
	state := randString()
	pkceVerifier := oauth2.GenerateVerifier()
	codeChallenge := oauth2.S256ChallengeOption(pkceVerifier)
	authorizeUrl := p.oauth.AuthCodeURL(state, codeChallenge)

	SetOIDCFlowCookie(w, state, pkceVerifier)
	http.Redirect(w, r, authorizeUrl, http.StatusFound)
}

func (p *Provider) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie("oidc_flow")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Failed to verify authentication cookie", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	cookieState, cookieVerifier, ok := strings.Cut(cookie.Value, ":")
	if !ok {
		http.Error(w, "failed to parse cookie value", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("state") != cookieState {
		http.Error(w, "state mismatch", http.StatusBadRequest)
	}

	token, err := p.oauth.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(cookieVerifier))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token", http.StatusBadGateway)
		return
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = claims
	_ = rawIDToken

}

func (p *Provider) FreshToken(ctx context.Context, stored *oauth2.Token) (*oauth2.Token, error) {
	return p.oauth.TokenSource(ctx, stored).Token()
}
