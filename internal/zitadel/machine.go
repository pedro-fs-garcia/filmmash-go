package zitadel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const tokenRefreshSkew = 60 * time.Second

type MachineAccessToken = string

type MachineTokenSource struct {
	logger       *slog.Logger
	httpClient   *http.Client
	tokenURL     string
	clientID     string
	clientSecret string
	scopes       []string

	mu          sync.Mutex
	accessToken MachineAccessToken
	expiry      time.Time
}

func NewMachineTokenSource(logger *slog.Logger, providerURL, clientID, clientSecret string) *MachineTokenSource {
	return &MachineTokenSource{
		logger:       logger,
		httpClient:   http.DefaultClient,
		tokenURL:     fmt.Sprintf("%s/oauth/v2/token", providerURL),
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       []string{"openid", "urn:zitadel:iam:org:project:id:zitadel:aud"},
	}
}

func (mts *MachineTokenSource) FetchAccessToken(ctx context.Context) (MachineAccessToken, error) {
	mts.mu.Lock()
	defer mts.mu.Unlock()

	if mts.accessToken != "" && time.Now().Before(mts.expiry.Add(-tokenRefreshSkew)) {
		return mts.accessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", strings.Join(mts.scopes, " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mts.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(mts.clientID, mts.clientSecret)

	resp, err := mts.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("zitadel token endpoint: status %d: %s", resp.StatusCode, body)
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("zitadel token endpoint: empty access_token")
	}

	mts.accessToken = tr.AccessToken
	mts.expiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return mts.accessToken, nil
}
