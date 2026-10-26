package library

import (
	"errors"
	"testing"
)

func TestNewShelfInvalid(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		capacity int
	}{
		{"capacity zero", "A", 0},
		{"negative capacity", "A", -1},
		{"empty id", "", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewShelf(tt.id, tt.capacity); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("NewShelf(%q, %d) error = %v, want %v", tt.id, tt.capacity, err, ErrInvalidInput)
			}
		})
	}
}

func TestShelfAdd(t *testing.T) {
	s, err := NewShelf("A", 2)
	if err != nil {
		t.Fatalf("NewShelf() error = %v", err)
	}
	b := mustBook(t, "Der Hobbit", "J. R. R. Tolkien", "9783608938004", 1937)

	for i := range 2 {
		if !s.HasSpace() {
			t.Fatalf("HasSpace() = false before adding book %d", i+1)
		}
		if err := s.Add(b); err != nil {
			t.Fatalf("Add() book %d error = %v", i+1, err)
		}
	}
	if s.HasSpace() {
		t.Error("HasSpace() = true on full shelf")
	}
	if err := s.Add(b); !errors.Is(err, ErrShelfFull) {
		t.Errorf("Add() on full shelf error = %v, want %v", err, ErrShelfFull)
	}
}

func mustBook(t *testing.T, title, author string, isbn ISBN, year int) Book {
	t.Helper()
	b, err := NewBook(title, author, isbn, year)
	if err != nil {
		t.Fatalf("NewBook() error = %v", err)
	}
	return b
}
