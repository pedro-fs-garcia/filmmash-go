package auth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

type ZitadelProvider struct {
	clientId     string
	clientSecret string
	providerUrl  string
}

func NewZitadelProvider(clientId, clientSecret, providerUrl string) *ZitadelProvider {
	return &ZitadelProvider{
		clientId:     clientId,
		clientSecret: clientSecret,
		providerUrl:  providerUrl,
	}
}

func (p *ZitadelProvider) VerifyIdToken(ctx context.Context, token string) (*oidc.IDToken, error) {
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
