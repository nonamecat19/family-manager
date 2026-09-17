package crypto

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const key = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func box(t *testing.T) *Box {
	t.Helper()
	b, err := NewBox(key)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRoundTrip(t *testing.T) {
	b := box(t)
	owner := Owner("fam", "user")
	sealed, err := b.Seal("1//refresh-token", owner)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("refresh-token")) {
		t.Fatal("plaintext visible in ciphertext")
	}
	got, err := b.Open(sealed, owner)
	if err != nil || got != "1//refresh-token" {
		t.Fatalf("Open = %q, %v", got, err)
	}
}

func TestSealIsRandomised(t *testing.T) {
	b := box(t)
	a, _ := b.Seal("same", Owner("f", "u"))
	c, _ := b.Seal("same", Owner("f", "u"))
	if bytes.Equal(a, c) {
		t.Fatal("two seals of the same plaintext must differ")
	}
}

func TestCiphertextIsBoundToItsOwner(t *testing.T) {
	b := box(t)
	sealed, _ := b.Seal("token", Owner("fam", "alice"))
	if _, err := b.Open(sealed, Owner("fam", "bob")); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("a token copied to another user's row must not open: %v", err)
	}
}

func TestTamperedOrShortCiphertext(t *testing.T) {
	b := box(t)
	sealed, _ := b.Seal("token", Owner("f", "u"))
	sealed[len(sealed)-1] ^= 1
	if _, err := b.Open(sealed, Owner("f", "u")); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("tampered: %v", err)
	}
	if _, err := b.Open([]byte{1, 2}, Owner("f", "u")); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("short: %v", err)
	}
}

func TestWrongKeyCannotOpen(t *testing.T) {
	sealed, _ := box(t).Seal("token", Owner("f", "u"))
	other, _ := NewBox("ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA=")
	if _, err := other.Open(sealed, Owner("f", "u")); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("wrong key: %v", err)
	}
}

func TestKeyValidation(t *testing.T) {
	for _, k := range []string{"", "not base64!", "c2hvcnQ="} {
		if _, err := NewBox(k); err == nil {
			t.Errorf("NewBox(%q) accepted a bad key", k)
		}
	}
	_, err := NewBox("c2hvcnQ=")
	if err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Fatalf("error should name the size: %v", err)
	}
}

func TestSealedBlobIsVersioned(t *testing.T) {
	b := box(t)
	sealed, _ := b.Seal("token", Owner("f", "u"))
	if sealed[0] != version1 {
		t.Fatalf("first byte = %d, want version %d", sealed[0], version1)
	}
	sealed[0] = 9
	if _, err := b.Open(sealed, Owner("f", "u")); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("unknown version must not open: %v", err)
	}
}

func TestOwnerBindsFamilyAndIsUnambiguous(t *testing.T) {
	b := box(t)
	sealed, _ := b.Seal("token", Owner("famA", "u"))
	if _, err := b.Open(sealed, Owner("famB", "u")); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("another family must not open it: %v", err)
	}
	if bytes.Equal(Owner("a:b", "c"), Owner("a", "b:c")) {
		t.Fatal("owner encoding must be unambiguous")
	}
	if _, err := b.Open(sealed, nil); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("empty owner: %v", err)
	}
}

func TestEmptyPlaintextAndMinimalBlobs(t *testing.T) {
	b := box(t)
	sealed, err := b.Seal("", Owner("f", "u"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := b.Open(sealed, Owner("f", "u")); err != nil || got != "" {
		t.Fatalf("empty round trip: %q %v", got, err)
	}
	if _, err := b.Open(sealed[:13], Owner("f", "u")); !errors.Is(err, ErrCiphertext) {
		t.Fatalf("version+nonce without a tag: %v", err)
	}
}
