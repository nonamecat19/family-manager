package auth

import "strings"

const MaxEmailLength = 254

func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func LooksLikeEmail(s string) bool {
	if s == "" || len(s) > MaxEmailLength {
		return false
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 || strings.Count(s, "@") != 1 {
		return false
	}
	domain := s[at+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") &&
		!strings.HasSuffix(domain, ".") && !strings.ContainsAny(s, " \t\r\n")
}
