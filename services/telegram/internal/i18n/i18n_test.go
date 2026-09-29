package i18n

import "testing"

func TestEveryKeyIsTranslatedInEveryLocale(t *testing.T) {
	for _, locale := range Supported {
		dict, ok := catalog[locale]
		if !ok {
			t.Fatalf("locale %q has no dictionary", locale)
		}
		for key := range catalog[Default] {
			if _, ok := dict[key]; !ok {
				t.Errorf("locale %q is missing key %q", locale, key)
			}
		}
		for key := range dict {
			if _, ok := catalog[Default][key]; !ok {
				t.Errorf("locale %q has an extra key %q", locale, key)
			}
		}
	}
}

func TestPlaceholdersMatchAcrossLocales(t *testing.T) {
	count := func(s string) int {
		var n int
		for i := 0; i < len(s)-1; i++ {
			if s[i] == '%' {
				if s[i+1] == '%' {
					i++
					continue
				}
				n++
			}
		}
		return n
	}

	for key, base := range catalog[Default] {
		for _, locale := range Supported {
			if got, want := count(catalog[locale][key]), count(base); got != want {
				t.Errorf("key %q: locale %q has %d placeholders, default has %d", key, locale, got, want)
			}
		}
	}
}

func TestMatchFallsBackToTheDefault(t *testing.T) {
	cases := map[string]Locale{
		"uk": UK, "uk-UA": UK, "en": EN, "en-GB": EN, "fr": Default, "": Default,
	}
	for tag, want := range cases {
		if got := Match(tag); got != want {
			t.Errorf("Match(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestTFallsBackWhenAKeyIsMissing(t *testing.T) {
	if got := T(EN, "nope.nope"); got != "nope.nope" {
		t.Fatalf("T for a missing key = %q", got)
	}
	if got := T("fr", FinanceBalance); got != T(Default, FinanceBalance) {
		t.Fatalf("unknown locale did not fall back: %q", got)
	}
}
