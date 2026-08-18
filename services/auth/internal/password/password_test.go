package password

import (
	"errors"
	"strings"
	"testing"
)

// Cheap parameters: these tests exercise the encoding and comparison, not the KDF's cost.
func testParams() Params {
	p := DefaultParams()
	p.Memory = 64
	p.Iterations = 1
	return p
}

func TestHashVerifyRoundTrip(t *testing.T) {
	encoded, err := Hash("correct horse battery staple", testParams())
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := Verify("correct horse battery staple", encoded); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyRejectsTheWrongPassword(t *testing.T) {
	encoded, err := Hash("right", testParams())
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := Verify("wrong", encoded); !errors.Is(err, ErrMismatch) {
		t.Fatalf("err = %v, want ErrMismatch", err)
	}
}

func TestHashIsSaltedPerCall(t *testing.T) {
	a, _ := Hash("same", testParams())
	b, _ := Hash("same", testParams())
	if a == b {
		t.Fatal("two hashes of the same password are identical — the salt is not random")
	}
	// Both must still verify.
	if err := Verify("same", a); err != nil {
		t.Errorf("first hash: %v", err)
	}
	if err := Verify("same", b); err != nil {
		t.Errorf("second hash: %v", err)
	}
}

func TestEncodedFormIsPHC(t *testing.T) {
	encoded, err := Hash("x", testParams())
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Errorf("encoded form = %q", encoded)
	}
	if got := len(strings.Split(encoded, "$")); got != 6 {
		t.Errorf("got %d $-separated fields, want 6", got)
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	for _, bad := range []string{
		"",
		"not-a-hash",
		"$argon2i$v=19$m=64,t=1,p=1$c2FsdA$aGFzaA",  // wrong algorithm
		"$argon2id$v=18$m=64,t=1,p=1$c2FsdA$aGFzaA", // wrong version
		"$argon2id$v=19$m=64,t=1$c2FsdA$aGFzaA",     // missing parallelism
		"$argon2id$v=19$m=64,t=1,p=1$!!!!$aGFzaA",   // salt is not base64
	} {
		if err := Verify("x", bad); !errors.Is(err, ErrBadHash) {
			t.Errorf("Verify(%q) = %v, want ErrBadHash", bad, err)
		}
	}
}

func TestNeedsRehashDetectsWeakerParameters(t *testing.T) {
	weak := testParams()
	encoded, err := Hash("x", weak)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	if NeedsRehash(encoded, weak) {
		t.Error("a hash written with the current parameters should not need a rehash")
	}

	stronger := weak
	stronger.Memory = weak.Memory * 4
	if !NeedsRehash(encoded, stronger) {
		t.Error("a hash written with less memory should need a rehash")
	}

	// An unreadable hash always needs replacing.
	if !NeedsRehash("garbage", weak) {
		t.Error("an unparseable hash should need a rehash")
	}
}
