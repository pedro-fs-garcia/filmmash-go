package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
)

func GenerateAES256Key() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func loadAESKey() ([]byte, error) {
	if b64 := os.Getenv("AES_KEY"); b64 != "" {
		key, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("AES_KEY is not valid base64: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("AES_KEY must decode to 32 bytes, got %d", len(key))
		}
		return key, nil
	}
	slog.Warn(
		`AES_KEY not set; using an ephemeral key.
		In-flight logins will not survive restarts or work across replicas.
		Set AES_KEY to a base64-encoded 32-byte key (e.g. 'openssl rand -base64 32')`,
	)
	return GenerateAES256Key()
}
