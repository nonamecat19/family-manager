package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

type Box struct {
	aead cipher.AEAD
}

var ErrCiphertext = errors.New("secret: malformed ciphertext")

func NewBox(keyB64 string) (*Box, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		key, err = base64.RawURLEncoding.DecodeString(keyB64)
	}
	if err != nil {
		return nil, fmt.Errorf("secret: key must be base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("secret: key must decode to 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secret: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secret: new gcm: %w", err)
	}
	return &Box{aead: aead}, nil
}

func (b *Box) Seal(plaintext string) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("secret: read nonce: %w", err)
	}
	return b.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

func (b *Box) Open(sealed []byte) (string, error) {
	if len(sealed) < b.aead.NonceSize() {
		return "", ErrCiphertext
	}
	nonce, body := sealed[:b.aead.NonceSize()], sealed[b.aead.NonceSize():]
	plaintext, err := b.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCiphertext, err)
	}
	return string(plaintext), nil
}
