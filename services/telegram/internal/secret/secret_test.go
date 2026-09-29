package secret

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
)

func newTestBox(t *testing.T) *Box {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("read key: %v", err)
	}
	box, err := NewBox(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	return box
}

func TestSealOpenRoundTrip(t *testing.T) {
	box := newTestBox(t)

	sealed, err := box.Seal("refresh-token")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if string(sealed) == "refresh-token" {
		t.Fatal("ciphertext is the plaintext")
	}

	opened, err := box.Open(sealed)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if opened != "refresh-token" {
		t.Fatalf("opened = %q, want %q", opened, "refresh-token")
	}
}

func TestSealUsesAFreshNonce(t *testing.T) {
	box := newTestBox(t)

	first, err := box.Seal("same")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	second, err := box.Seal("same")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if string(first) == string(second) {
		t.Fatal("two seals of the same plaintext are identical")
	}
}

func TestOpenRejectsTamperedCiphertext(t *testing.T) {
	box := newTestBox(t)

	sealed, err := box.Seal("refresh-token")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	sealed[len(sealed)-1] ^= 0xff

	if _, err := box.Open(sealed); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("err = %v, want ErrCiphertext", err)
	}
}

func TestOpenRejectsAForeignKey(t *testing.T) {
	sealed, err := newTestBox(t).Seal("refresh-token")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if _, err := newTestBox(t).Open(sealed); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("err = %v, want ErrCiphertext", err)
	}
}

func TestNewBoxRejectsAShortKey(t *testing.T) {
	if _, err := NewBox(base64.StdEncoding.EncodeToString([]byte("too short"))); err == nil {
		t.Fatal("NewBox accepted a key that is not 32 bytes")
	}
}
