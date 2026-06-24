package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Zitadel struct {
	clientId     string
	clientSecret string
	providerUrl  string
}

func NewZitadelProvider(clientId, clientSecret, providerUrl string) *Zitadel {
	return &Zitadel{
		clientId:     clientId,
		clientSecret: clientSecret,
		providerUrl:  providerUrl,
	}
}

func (p *Zitadel) VerifyIdToken(ctx context.Context, token string) (*oidc.IDToken, error) {
	provider, err := oidc.NewProvider(ctx, p.providerUrl)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: p.clientId})
	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	return idToken, nil
}

func (p *Zitadel) RequestToken(code, verifier string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:8080/auth/callback")
	form.Set("code_verifier", verifier)

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/oauth/v2/token", p.providerUrl),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return TokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.SetBasicAuth(p.clientId, p.clientSecret)
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

func (p *Zitadel) AuthorizeURL(state, codeChallenge string) (*url.URL, error) {
	u, err := url.Parse(fmt.Sprintf("%s/oauth/v2/authorize", p.providerUrl))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("client_id", p.clientId)
	q.Set("redirect_uri", "http://localhost:8080/auth/callback")
	q.Set("response_type", "code")
	q.Set("scope", "openid profile email offline_access")
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u, nil
}

func (p *Zitadel) EndSessionURL(idToken string) (*url.URL, error) {
	u, err := url.Parse(fmt.Sprintf("%s/oidc/v1/end_session", p.providerUrl))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("id_token_hint", idToken)
	q.Set("post_logout_redirect_uri", "http://localhost:8080/ui/films")
	u.RawQuery = q.Encode()
	return u, nil
}
