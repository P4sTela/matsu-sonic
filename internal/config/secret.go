package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// encPrefix marks a value as AES-GCM encrypted (followed by base64 ciphertext).
const encPrefix = "enc:"

// keyFileName is the name of the local secret key file, stored alongside config.
const keyFileName = "secret.key"

// loadOrCreateKey returns the 32-byte encryption key stored in dir/secret.key,
// generating and persisting a new one if it does not yet exist.
func loadOrCreateKey(dir string) ([]byte, error) {
	keyPath := filepath.Join(dir, keyFileName)
	if data, err := os.ReadFile(keyPath); err == nil && len(data) == 32 {
		return data, nil
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}
	return key, nil
}

// encryptValue encrypts plaintext with AES-GCM and returns an "enc:"-prefixed,
// base64-encoded string. Empty and already-encrypted values are returned as-is.
func encryptValue(key []byte, plaintext string) (string, error) {
	if plaintext == "" || strings.HasPrefix(plaintext, encPrefix) {
		return plaintext, nil
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ct), nil
}

// decryptValue reverses encryptValue. Values without the "enc:" prefix are
// assumed to be plaintext (backward compatibility) and returned unchanged.
func decryptValue(key []byte, value string) (string, error) {
	if !strings.HasPrefix(value, encPrefix) {
		return value, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encPrefix))
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plaintext), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// encryptSecrets returns a copy of cfg with sensitive fields encrypted at rest.
func encryptSecrets(cfg Config, dir string) (Config, error) {
	if len(cfg.DistTargets) == 0 {
		return cfg, nil
	}
	key, err := loadOrCreateKey(dir)
	if err != nil {
		return cfg, err
	}
	cfg.DistTargets = append([]DistTargetConf(nil), cfg.DistTargets...)
	for i := range cfg.DistTargets {
		enc, err := encryptValue(key, cfg.DistTargets[i].Password)
		if err != nil {
			return cfg, err
		}
		cfg.DistTargets[i].Password = enc
	}
	return cfg, nil
}

// decryptSecrets decrypts sensitive fields in cfg in place.
func decryptSecrets(cfg *Config, dir string) error {
	if len(cfg.DistTargets) == 0 {
		return nil
	}
	// Only touch the key file if there is something encrypted to decrypt.
	hasEncrypted := false
	for _, t := range cfg.DistTargets {
		if strings.HasPrefix(t.Password, encPrefix) {
			hasEncrypted = true
			break
		}
	}
	if !hasEncrypted {
		return nil
	}
	key, err := loadOrCreateKey(dir)
	if err != nil {
		return err
	}
	for i := range cfg.DistTargets {
		dec, err := decryptValue(key, cfg.DistTargets[i].Password)
		if err != nil {
			return err
		}
		cfg.DistTargets[i].Password = dec
	}
	return nil
}
