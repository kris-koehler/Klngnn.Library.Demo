package main

import (
	"fmt"

	"buecherei/internal/library"
)

type sampleBook struct {
	title, author, isbn string
	year                int
	referenceOnly       bool
}

// seed creates shelves A and B with two slots each and adds the sample books.
func seed(lib *library.Library) error {
	for _, id := range []string{"A", "B"} {
		if err := lib.AddShelf(id, 2); err != nil {
			return err
		}
	}

	samples := []sampleBook{
		{"Der Hobbit", "J. R. R. Tolkien", "978-3-608-93800-4", 1937, false},
		{"Die Tribute von Panem", "Suzanne Collins", "978-3-7891-3218-5", 2008, false},
		// Assumption for demonstration purposes (see README).
		{"Clean Code", "Robert C. Martin", "978-0-13-235088-4", 2008, true},
	}
	for _, s := range samples {
		isbn, err := library.ParseISBN(s.isbn)
		if err != nil {
			return err
		}
		newBook := library.NewBook
		if s.referenceOnly {
			newBook = library.NewReferenceBook
		}
		b, err := newBook(s.title, s.author, isbn, s.year)
		if err != nil {
			return err
		}
		if _, err := lib.AddBook(b); err != nil {
			return fmt.Errorf("add %q: %w", s.title, err)
		}
	}
	return nil
}
