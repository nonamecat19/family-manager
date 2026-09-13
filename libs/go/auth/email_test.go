package auth

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"  Ada@Example.Test ": "ada@example.test",
		"ada@example.test":    "ada@example.test",
		"":                    "",
	}
	for in, want := range cases {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeEmailMakesTheTwoSidesAgree(t *testing.T) {
	invited := NormalizeEmail("Ada.Lovelace@Example.Test")
	registered := NormalizeEmail("ada.lovelace@example.test  ")
	if invited != registered {
		t.Fatalf("%q != %q", invited, registered)
	}
}

func TestLooksLikeEmail(t *testing.T) {
	valid := []string{"a@b.co", "ada.lovelace+tag@example.co.uk", "ада@приклад.укр"}
	for _, s := range valid {
		if !LooksLikeEmail(s) {
			t.Errorf("LooksLikeEmail(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"", "a", "a@", "@b.co", "a@b", "a b@c.co", "a@@b.co", "a@.co", "a@b.",
		"a@b.co\nBcc: someone@else.test",
		strings.Repeat("a", MaxEmailLength) + "@example.test",
	}
	for _, s := range invalid {
		if LooksLikeEmail(s) {
			t.Errorf("LooksLikeEmail(%q) = true, want false", s)
		}
	}
}
