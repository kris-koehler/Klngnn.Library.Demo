package library

import (
	"fmt"
	"strings"
)

// Book is an immutable value describing a single copy of a title.
// Loan state is tracked by the Library, not by the book itself.
type Book struct {
	Title         string
	Author        string
	ISBN          ISBN
	Year          int
	ReferenceOnly bool
}

// NewBook creates a book that can be lent.
func NewBook(title, author string, isbn ISBN, year int) (Book, error) {
	return newBook(title, author, isbn, year, false)
}

// NewReferenceBook creates a reference-only book that cannot be lent.
func NewReferenceBook(title, author string, isbn ISBN, year int) (Book, error) {
	return newBook(title, author, isbn, year, true)
}

func newBook(title, author string, isbn ISBN, year int, referenceOnly bool) (Book, error) {
	if strings.TrimSpace(title) == "" {
		return Book{}, fmt.Errorf("%w: title must not be empty", ErrInvalidInput)
	}
	if strings.TrimSpace(author) == "" {
		return Book{}, fmt.Errorf("%w: author must not be empty", ErrInvalidInput)
	}
	if year <= 0 {
		return Book{}, fmt.Errorf("%w: year must be positive, got %d", ErrInvalidInput, year)
	}
	// The zero value would otherwise bypass ParseISBN.
	if isbn == "" {
		return Book{}, fmt.Errorf("%w: isbn must not be empty", ErrInvalidInput)
	}
	return Book{
		Title:         title,
		Author:        author,
		ISBN:          isbn,
		Year:          year,
		ReferenceOnly: referenceOnly,
	}, nil
}
