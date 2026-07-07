package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"filmmash/internal/zitadel"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func generateNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

type LoginChallenge struct {
	AuthorizeURL string
	State        string
	Nonce        string
	Verifier     string
}

type Provider struct {
	provider      *oidc.Provider
	machineClient *zitadel.MachineTokenSource
	oauth         *oauth2.Config
	verifier      *oidc.IDTokenVerifier
	logoutURL     string
	issuer        string
	appBaseUrl    string
}

func NewProvider(ctx context.Context,
	machineClient *zitadel.MachineTokenSource, issuer, clientId, clientSecret, callbackURL, appBaseUrl string,
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
		provider:      provider,
		machineClient: machineClient,
		oauth: &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email", oidc.ScopeOfflineAccess},
		},
		verifier:   provider.Verifier(&oidc.Config{ClientID: clientId}),
		logoutURL:  meta.EndSessionEndpoint,
		issuer:     issuer,
		appBaseUrl: appBaseUrl,
	}, nil
}

func (p *Provider) NewLoginChallenge() (*LoginChallenge, error) {
	state := randString()
	verifier := oauth2.GenerateVerifier()
	nonce, err := generateNonce()
	if err != nil {
		return nil, err

	}
	authURL := p.oauth.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
		oidc.Nonce(nonce),
	)

	return &LoginChallenge{
		AuthorizeURL: authURL,
		State:        state,
		Nonce:        nonce,
		Verifier:     verifier,
	}, nil
}

func (p *Provider) EndSessionURL(idToken string) (*url.URL, error) {
	u, err := url.Parse(p.logoutURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("id_token_hint", idToken)
	q.Set("post_logout_redirect_uri", p.appBaseUrl+"/ui/films")
	u.RawQuery = q.Encode()
	return u, nil
}

func (p *Provider) RequestToken(ctx context.Context, code, verifier, nonce string) (*AuthTokens, error) {
	token, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("token response missing id_token")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	if idToken.Nonce != nonce {
		return nil, errors.New("nonce mismatch")
	}

	return &AuthTokens{
		Token:      token,
		IDToken:    idToken,
		RawIDToken: rawIDToken,
	}, nil
}

func (p *Provider) GetUserInfo(ctx context.Context, accessToken string) (UserInfo, error) {
	info, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}))
	if err != nil {
		return UserInfo{}, fmt.Errorf("fetching userinfo: %w", err)
	}

	var raw UserInfo
	if err := info.Claims(&raw); err != nil {
		return UserInfo{}, fmt.Errorf("decoding userinfo: %w", err)
	}
	return raw, nil
}

func (p *Provider) FetchAuthorizations(ctx context.Context, subs []string) (map[string]*UserAuthz, error) {
	body, err := json.Marshal(map[string]any{
		"filters": []map[string]any{
			{"inUserIds": map[string]any{"ids": subs}},
		},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		p.issuer+"/zitadel.authorization.v2.AuthorizationService/ListAuthorizations",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	accessToken, err := p.machineClient.FetchAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Connect-Protocol-Version", "1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("zitadel ListAuthorizations: status %d: %s", resp.StatusCode, respBody)
	}

	var page zitadel.ListAuthzResp
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	out := make(map[string]*UserAuthz, len(subs))
	for _, a := range page.Authorizations {
		u := &UserAuthz{
			UserID:      a.User.ID,
			LoginName:   a.User.PreferredLoginName,
			DisplayName: a.User.DisplayName,
		}
		for _, r := range a.Roles {
			u.Roles = append(u.Roles, r.DisplayName)
		}
		out[a.User.ID] = u
	}

	return out, nil
}
