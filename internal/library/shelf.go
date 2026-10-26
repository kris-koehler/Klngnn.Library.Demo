package library

import (
	"fmt"
	"strings"
)

// Shelf holds a limited number of books. It is not safe for concurrent use;
// the Library guards all access to its shelves.
type Shelf struct {
	id       string
	capacity int
	books    []Book
}

// NewShelf creates an empty shelf with the given ID and capacity.
func NewShelf(id string, capacity int) (*Shelf, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: shelf id must not be empty", ErrInvalidInput)
	}
	if capacity <= 0 {
		return nil, fmt.Errorf("%w: shelf capacity must be positive, got %d", ErrInvalidInput, capacity)
	}
	return &Shelf{id: id, capacity: capacity, books: make([]Book, 0, capacity)}, nil
}

// ID returns the shelf's identifier.
func (s *Shelf) ID() string {
	return s.id
}

// HasSpace reports whether another book fits on the shelf.
func (s *Shelf) HasSpace() bool {
	return len(s.books) < s.capacity
}

// Add places a book on the shelf.
func (s *Shelf) Add(b Book) error {
	if !s.HasSpace() {
		return fmt.Errorf("%w: shelf %q", ErrShelfFull, s.id)
	}
	s.books = append(s.books, b)
	return nil
}
