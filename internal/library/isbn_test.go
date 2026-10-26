package library

import (
	"errors"
	"testing"
)

func TestParseISBN(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    ISBN
		wantErr error
	}{
		{"with hyphens", "978-3-608-93800-4", "9783608938004", nil},
		{"digits only", "9783608938004", "9783608938004", nil},
		{"with spaces", "978 3 608 93800 4", "9783608938004", nil},
		{"empty", "", "", ErrInvalidISBN},
		{"too short", "978-3-608", "", ErrInvalidISBN},
		{"too long", "97836089380045", "", ErrInvalidISBN},
		{"non-digit", "978-3-608-93800-X", "", ErrInvalidISBN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseISBN(tt.raw)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseISBN(%q) error = %v, want %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseISBN(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
