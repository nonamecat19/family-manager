// Package password hashes and verifies passwords with argon2id.
//
// It lives in services/auth and nowhere else (docs/adr/0005-auth.md): no other service ever
// sees a password, so no other service needs this code. The encoded form is the standard PHC
// string, so the parameters travel with the hash and can be raised later without invalidating
// existing ones.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params are the argon2id cost parameters. The defaults follow OWASP's 2024 guidance for
// argon2id (19 MiB, 2 iterations, 1 lane).
type Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultParams() Params {
	return Params{
		Memory:      19 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}

var (
	// ErrMismatch is returned when the password does not match the hash. It is deliberately
	// indistinguishable from "no such user" at the handler level.
	ErrMismatch = errors.New("password: does not match")
	// ErrBadHash means the stored value is not a hash this package wrote.
	ErrBadHash = errors.New("password: unrecognised hash format")
)

// Hash returns a PHC-encoded argon2id hash:
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
func Hash(plaintext string, p Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: read salt: %w", err)
	}

	sum := argon2.IDKey([]byte(plaintext), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	), nil
}

// Verify recomputes the hash with the parameters recorded in the encoded value and compares
// in constant time — a byte-by-byte comparison would leak the prefix length through timing.
func Verify(plaintext, encoded string) error {
	p, salt, want, err := decode(encoded)
	if err != nil {
		return err
	}

	got := argon2.IDKey([]byte(plaintext), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrMismatch
	}
	return nil
}

// NeedsRehash reports whether a stored hash was written with weaker parameters than the ones
// now in force, so a successful login can transparently upgrade it.
func NeedsRehash(encoded string, p Params) bool {
	stored, _, _, err := decode(encoded)
	if err != nil {
		return true
	}
	return stored.Memory < p.Memory ||
		stored.Iterations < p.Iterations ||
		stored.KeyLength < p.KeyLength
}

func decode(encoded string) (p Params, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, hash]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, ErrBadHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, ErrBadHash
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return p, nil, nil, ErrBadHash
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return p, nil, nil, ErrBadHash
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return p, nil, nil, ErrBadHash
	}

	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(hash))
	return p, salt, hash, nil
}
