package library

import (
	"fmt"
	"strings"
)

// ISBN is a normalized ISBN-13 consisting of exactly 13 ASCII digits.
type ISBN string

// ParseISBN removes hyphens and spaces and requires exactly 13 digits.
// The check digit is deliberately not validated (see README).
func ParseISBN(raw string) (ISBN, error) {
	s := normalizeISBN(raw)
	if len(s) != 13 {
		return "", fmt.Errorf("%w: %q", ErrInvalidISBN, raw)
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("%w: %q", ErrInvalidISBN, raw)
		}
	}
	return ISBN(s), nil
}

// normalizeISBN removes hyphens and spaces.
func normalizeISBN(raw string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(raw)
}
