package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const cookieName = "__Host-oidc_flow"

var ErrInvalidFlowCookie = errors.New("invalid oidc flow cookie")

type oidcFlowPayload struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Exp      int64  `json:"e"`
}

type OIDCFlowCodec struct {
	aead cipher.AEAD
}

func NewOIDCFlowCodec(key []byte) (*OIDCFlowCodec, error) {
	if len(key) != 32 {
		return nil, errors.New("oidc flow codec: key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &OIDCFlowCodec{aead: aead}, nil
}

func (c *OIDCFlowCodec) SetCookie(w http.ResponseWriter, state, nonce, verifier string) error {
	payload, err := json.Marshal(oidcFlowPayload{
		State:    state,
		Nonce:    nonce,
		Verifier: verifier,
		Exp:      time.Now().Add(5 * time.Minute).Unix(),
	})
	if err != nil {
		return err
	}

	iv := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return err
	}
	sealed := c.aead.Seal(iv, iv, payload, nil) // output: iv || ciphertext || tag

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    base64.RawURLEncoding.EncodeToString(sealed),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   300,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (c *OIDCFlowCodec) ReadCookie(r *http.Request) (state, nonce, verifier string, err error) {
	ck, err := r.Cookie(cookieName)
	if err != nil {
		return "", "", "", ErrInvalidFlowCookie
	}
	raw, err := base64.RawURLEncoding.DecodeString(ck.Value)
	if err != nil || len(raw) < c.aead.NonceSize() {
		return "", "", "", ErrInvalidFlowCookie
	}
	pt, err := c.aead.Open(nil, raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():], nil)
	if err != nil {
		return "", "", "", ErrInvalidFlowCookie
	}
	var p oidcFlowPayload
	if err := json.Unmarshal(pt, &p); err != nil {
		return "", "", "", ErrInvalidFlowCookie
	}
	if time.Now().Unix() > p.Exp {
		return "", "", "", ErrInvalidFlowCookie
	}
	return p.State, p.Nonce, p.Verifier, nil
}

func ClearOIDCFlowCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}
