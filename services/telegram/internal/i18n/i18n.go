package i18n

import (
	"fmt"
	"strings"
)

type Locale string

const (
	UK      Locale = "uk"
	EN      Locale = "en"
	Default        = UK
)

var Supported = []Locale{UK, EN}

type Key string

func Match(tag string) Locale {
	lower := strings.ToLower(strings.TrimSpace(tag))
	for _, locale := range Supported {
		if strings.HasPrefix(lower, string(locale)) {
			return locale
		}
	}
	return Default
}

func (l Locale) Name() string {
	switch l {
	case EN:
		return "English"
	case UK:
		return "Українська"
	default:
		return string(l)
	}
}

func (l Locale) Flag() string {
	switch l {
	case EN:
		return "🇬🇧"
	case UK:
		return "🇺🇦"
	default:
		return "🏳"
	}
}

func T(locale Locale, key Key, args ...any) string {
	dict, ok := catalog[locale]
	if !ok {
		dict = catalog[Default]
	}
	format, ok := dict[key]
	if !ok {
		if format, ok = catalog[Default][key]; !ok {
			return string(key)
		}
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

type Printer struct{ locale Locale }

func For(locale Locale) Printer { return Printer{locale: locale} }

func (p Printer) T(key Key, args ...any) string { return T(p.locale, key, args...) }

func (p Printer) Locale() Locale { return p.locale }
