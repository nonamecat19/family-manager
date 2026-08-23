package auth

import "strings"

// MaxEmailLength is RFC 5321's practical ceiling on a full address: 64 octets of local part,
// an @, and 255 of domain, capped at 254 by the SMTP path length. Anything longer is not an
// address mail could be delivered to.
const MaxEmailLength = 254

// NormalizeEmail is the canonical form an address is stored and compared in: trimmed and
// lowercased.
//
// It lives here rather than in services/auth because it is not only that service's concern.
// An invitation in services/family is addressed to a person who will sign in through
// services/auth, and if the two disagree about whether "Ada@Example.com" and
// "ada@example.com" are the same person, the invitation is for an account that does not exist.
//
// Only the domain is case-insensitive by RFC; the local part is not. Lowercasing both anyway
// is the choice every mail provider a family uses has already made, and treating
// Ada@ and ada@ as different people would produce a duplicate-account bug far more often than
// it would ever serve someone who genuinely has both.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// LooksLikeEmail is a sanity check, not a validator. RFC 5322 permits addresses no regex
// should try to describe; the real proof that an address works is that mail to it arrives.
// This rejects the shapes that are certainly wrong — no @, nothing before or after it, more
// than one, a domain with no dot or a leading/trailing one, embedded whitespace — and the
// length ceiling above.
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
