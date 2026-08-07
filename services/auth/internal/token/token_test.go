package token

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
)

func testKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func testSigner(t *testing.T) *Signer {
	t.Helper()
	s, err := NewSigner(Config{
		PrivateKeyPEM: testKeyPEM(t),
		Issuer:        "family-manager",
		Audience:      "family-manager",
		TTL:           15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return s
}

// The contract that matters: a token this package mints must verify with the library every
// other service uses. If these two ever drift, every request in the system fails.
func TestTokenVerifiesWithLibsGoAuth(t *testing.T) {
	signer := testSigner(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(signer.JWKS())
	}))
	defer srv.Close()

	verifier, err := fmauth.NewVerifier(fmauth.VerifierConfig{
		JWKSURL:  srv.URL,
		Issuer:   "family-manager",
		Audience: "family-manager",
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	signed, err := signer.Sign(Claims{UserID: "u1", Email: "a@b.test", FamilyID: "f1"})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	claims, err := verifier.Verify(context.Background(), signed)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != "u1" || claims.Email != "a@b.test" || claims.FamilyID != "f1" {
		t.Errorf("claims = %+v", claims)
	}
}

func TestFamilylessTokenHasNoFamilyClaim(t *testing.T) {
	signer := testSigner(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(signer.JWKS())
	}))
	defer srv.Close()

	verifier, _ := fmauth.NewVerifier(fmauth.VerifierConfig{JWKSURL: srv.URL})
	signed, err := signer.Sign(Claims{UserID: "u1"})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	claims, err := verifier.Verify(context.Background(), signed)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.FamilyID != "" {
		t.Errorf("FamilyID = %q, want empty", claims.FamilyID)
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	signer, err := NewSigner(Config{
		PrivateKeyPEM: testKeyPEM(t),
		TTL:           time.Minute,
		Now:           func() time.Time { return past },
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(signer.JWKS())
	}))
	defer srv.Close()

	verifier, _ := fmauth.NewVerifier(fmauth.VerifierConfig{JWKSURL: srv.URL})
	signed, _ := signer.Sign(Claims{UserID: "u1"})

	if _, err := verifier.Verify(context.Background(), signed); err == nil {
		t.Fatal("an expired token verified")
	}
}

func TestTokenFromAnotherKeyIsRejected(t *testing.T) {
	ours := testSigner(t)
	theirs := testSigner(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(ours.JWKS())
	}))
	defer srv.Close()

	verifier, _ := fmauth.NewVerifier(fmauth.VerifierConfig{JWKSURL: srv.URL})
	forged, _ := theirs.Sign(Claims{UserID: "attacker"})

	if _, err := verifier.Verify(context.Background(), forged); err == nil {
		t.Fatal("a token signed by an unknown key verified")
	}
}

func TestIssuerAndAudienceAreEnforced(t *testing.T) {
	signer, err := NewSigner(Config{
		PrivateKeyPEM: testKeyPEM(t), Issuer: "somewhere-else", Audience: "other", TTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(signer.JWKS())
	}))
	defer srv.Close()

	verifier, _ := fmauth.NewVerifier(fmauth.VerifierConfig{
		JWKSURL: srv.URL, Issuer: "family-manager", Audience: "family-manager",
	})
	signed, _ := signer.Sign(Claims{UserID: "u1"})

	if _, err := verifier.Verify(context.Background(), signed); err == nil {
		t.Fatal("a token from the wrong issuer verified")
	}
}

func TestJWKSCarriesOnlyPublicMaterial(t *testing.T) {
	signer := testSigner(t)
	set := signer.JWKS()

	if len(set.Keys) != 1 {
		t.Fatalf("got %d keys, want 1", len(set.Keys))
	}
	k := set.Keys[0]
	if k.Kty != "EC" || k.Crv != "P-256" || k.Alg != "ES256" || k.Use != "sig" {
		t.Errorf("jwk = %+v", k)
	}
	if k.Kid == "" {
		t.Error("kid is empty; verifiers cannot select a key during rotation")
	}

	// Serialised, the set must not contain the private scalar under any field name.
	raw, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{`"d"`, "PRIVATE"} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("JWKS contains %s: %s", forbidden, raw)
		}
	}
}

func TestKeyIDIsStableAndKeySpecific(t *testing.T) {
	pemText := testKeyPEM(t)
	a, _ := NewSigner(Config{PrivateKeyPEM: pemText})
	b, _ := NewSigner(Config{PrivateKeyPEM: pemText})
	if a.JWKS().Keys[0].Kid != b.JWKS().Keys[0].Kid {
		t.Error("the same key produced two different kids")
	}

	other := testSigner(t)
	if a.JWKS().Keys[0].Kid == other.JWKS().Keys[0].Kid {
		t.Error("two different keys produced the same kid")
	}
}

func TestNewSignerRejectsUnusableKeys(t *testing.T) {
	if _, err := NewSigner(Config{PrivateKeyPEM: "not pem"}); err == nil {
		t.Error("expected an error for non-PEM input")
	}

	// RSA is a perfectly good key and completely wrong here.
	rsaPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("nonsense")}))
	if _, err := NewSigner(Config{PrivateKeyPEM: rsaPEM}); err == nil {
		t.Error("expected an error for a non-EC key")
	}

	// P-384 is stronger and still wrong: ES256 means P-256.
	key, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	p384 := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if _, err := NewSigner(Config{PrivateKeyPEM: p384}); err == nil {
		t.Error("expected an error for a P-384 key")
	}
}

func TestSEC1KeysAreAccepted(t *testing.T) {
	// `openssl ecparam -genkey` emits SEC1, which is what an operator is most likely to have.
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	sec1 := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))

	if _, err := NewSigner(Config{PrivateKeyPEM: sec1}); err != nil {
		t.Fatalf("NewSigner with a SEC1 key: %v", err)
	}
}
