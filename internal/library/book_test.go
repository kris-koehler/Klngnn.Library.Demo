package library

import (
	"errors"
	"testing"
)

func TestNewBook(t *testing.T) {
	const isbn ISBN = "9783608938004"
	tests := []struct {
		name    string
		title   string
		author  string
		isbn    ISBN
		year    int
		wantErr error
	}{
		{"valid", "Der Hobbit", "J. R. R. Tolkien", isbn, 1937, nil},
		{"empty title", "", "J. R. R. Tolkien", isbn, 1937, ErrInvalidInput},
		{"blank author", "Der Hobbit", "   ", isbn, 1937, ErrInvalidInput},
		{"year zero", "Der Hobbit", "J. R. R. Tolkien", isbn, 0, ErrInvalidInput},
		{"empty isbn", "Der Hobbit", "J. R. R. Tolkien", "", 1937, ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBook(tt.title, tt.author, tt.isbn, tt.year)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewBook() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil && b.ReferenceOnly {
				t.Error("NewBook() returned a reference-only book")
			}
		})
	}
}

func TestNewReferenceBook(t *testing.T) {
	b, err := NewReferenceBook("Clean Code", "Robert C. Martin", "9780132350884", 2008)
	if err != nil {
		t.Fatalf("NewReferenceBook() error = %v", err)
	}
	if !b.ReferenceOnly {
		t.Error("NewReferenceBook() ReferenceOnly = false, want true")
	}
}
