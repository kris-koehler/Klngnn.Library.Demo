package library

import (
	"fmt"
	"strings"
	"sync"
)

// Loan records that a book is currently lent to a borrower.
type Loan struct {
	ISBN     ISBN
	Borrower string
}

// SearchResult describes a book found by Search.
type SearchResult struct {
	Book    Book
	ShelfID string
	Lent    bool // deliberately without the borrower's name: the API is public (privacy)
}

// Library manages shelves, the books on them and their loans.
// It is safe for concurrent use.
type Library struct {
	mu      sync.RWMutex
	shelves []*Shelf
	loans   map[ISBN]Loan
}

// New creates an empty library without shelves.
func New() *Library {
	return &Library{loans: make(map[ISBN]Loan)}
}

// AddShelf creates a new shelf. The library owns its shelves so that nobody
// outside can modify them without holding the lock.
func (l *Library) AddShelf(id string, capacity int) error {
	s, err := NewShelf(id, capacity)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, existing := range l.shelves {
		if existing.ID() == id {
			return fmt.Errorf("%w: shelf %q already exists", ErrInvalidInput, id)
		}
	}
	l.shelves = append(l.shelves, s)
	return nil
}

// AddBook places the book on the first shelf with free space and returns
// that shelf's ID.
func (l *Library) AddBook(b Book) (shelfID string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, _, ok := l.locate(b.ISBN); ok {
		return "", fmt.Errorf("%w: %s", ErrDuplicateISBN, b.ISBN)
	}
	for _, s := range l.shelves {
		if s.HasSpace() {
			if err := s.Add(b); err != nil {
				return "", err
			}
			return s.ID(), nil
		}
	}
	return "", ErrNoSpace
}

// Search returns all books whose title or author contains the query
// (case-insensitive) or whose ISBN contains the query without hyphens and
// spaces. An empty query returns all books. The result is never nil.
func (l *Library) Search(query string) []SearchResult {
	query = strings.TrimSpace(query)
	lowerQuery := strings.ToLower(query)
	isbnQuery := normalizeISBN(query)

	l.mu.RLock()
	defer l.mu.RUnlock()

	results := make([]SearchResult, 0)
	for _, s := range l.shelves {
		for _, b := range s.books {
			if !matches(b, lowerQuery, isbnQuery) {
				continue
			}
			_, lent := l.loans[b.ISBN]
			results = append(results, SearchResult{Book: b, ShelfID: s.ID(), Lent: lent})
		}
	}
	return results
}

func matches(b Book, lowerQuery, isbnQuery string) bool {
	if lowerQuery == "" {
		return true
	}
	if strings.Contains(strings.ToLower(b.Title), lowerQuery) ||
		strings.Contains(strings.ToLower(b.Author), lowerQuery) {
		return true
	}
	return isbnQuery != "" && strings.Contains(string(b.ISBN), isbnQuery)
}

// Lend lends the book with the given ISBN to the borrower. All checks and the
// update happen under a single lock, so concurrent calls cannot both succeed.
func (l *Library) Lend(isbn ISBN, borrower string) (Loan, error) {
	borrower = strings.TrimSpace(borrower)
	if borrower == "" {
		return Loan{}, fmt.Errorf("%w: borrower must not be empty", ErrInvalidInput)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	b, _, ok := l.locate(isbn)
	if !ok {
		return Loan{}, fmt.Errorf("%w: %s", ErrBookNotFound, isbn)
	}
	if b.ReferenceOnly {
		return Loan{}, fmt.Errorf("%w: %s", ErrReferenceOnly, isbn)
	}
	if _, lent := l.loans[isbn]; lent {
		return Loan{}, fmt.Errorf("%w: %s", ErrAlreadyLent, isbn)
	}

	loan := Loan{ISBN: isbn, Borrower: borrower}
	l.loans[isbn] = loan
	return loan, nil
}

// Return ends the loan of the book with the given ISBN. The book never left
// its shelf slot, so a return cannot fail because of a full shelf.
func (l *Library) Return(isbn ISBN) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, _, ok := l.locate(isbn); !ok {
		return fmt.Errorf("%w: %s", ErrBookNotFound, isbn)
	}
	if _, lent := l.loans[isbn]; !lent {
		return fmt.Errorf("%w: %s", ErrNotLent, isbn)
	}
	delete(l.loans, isbn)
	return nil
}

// locate expects l.mu to be held.
func (l *Library) locate(isbn ISBN) (Book, *Shelf, bool) {
	for _, s := range l.shelves {
		for _, b := range s.books {
			if b.ISBN == isbn {
				return b, s, true
			}
		}
	}
	return Book{}, nil, false
}
