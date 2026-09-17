package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrCiphertext = errors.New("crypto: malformed or foreign ciphertext")

const version1 byte = 1

type Box struct {
	aead cipher.AEAD
}

func NewBox(keyB64 string) (*Box, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("crypto: key must be base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}
	return &Box{aead: aead}, nil
}

func Owner(familyID, userID string) []byte {
	out := []byte("google-refresh-token")
	for _, part := range []string{familyID, userID} {
		out = binary.BigEndian.AppendUint32(out, uint32(len(part)))
		out = append(out, part...)
	}
	return out
}

func aad(version byte, owner []byte) []byte {
	return append([]byte{version}, owner...)
}

func (b *Box) Seal(plaintext string, owner []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: read nonce: %w", err)
	}
	return b.aead.Seal(append([]byte{version1}, nonce...), nonce, []byte(plaintext), aad(version1, owner)), nil
}

func (b *Box) Open(sealed []byte, owner []byte) (string, error) {
	if len(sealed) < 1+b.aead.NonceSize() || sealed[0] != version1 {
		return "", ErrCiphertext
	}
	sealed = sealed[1:]
	nonce, body := sealed[:b.aead.NonceSize()], sealed[b.aead.NonceSize():]
	plaintext, err := b.aead.Open(nil, nonce, body, aad(version1, owner))
	if err != nil {
		return "", ErrCiphertext
	}
	return string(plaintext), nil
}
