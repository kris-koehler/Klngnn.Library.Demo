package library

import "fmt"

// Snapshot is a copy of the complete library state, used for persistence.
type Snapshot struct {
	Shelves []ShelfSnapshot
	Loans   []Loan
}

// ShelfSnapshot is a copy of a single shelf and the books on it.
type ShelfSnapshot struct {
	ID       string
	Capacity int
	Books    []Book
}

// NewFromSnapshot rebuilds a library from a snapshot and validates it with
// the same rules that apply at runtime.
func NewFromSnapshot(snap Snapshot) (*Library, error) {
	l := New()
	for _, ss := range snap.Shelves {
		if err := l.AddShelf(ss.ID, ss.Capacity); err != nil {
			return nil, err
		}
		s := l.shelves[len(l.shelves)-1]
		for _, b := range ss.Books {
			if _, err := newBook(b.Title, b.Author, b.ISBN, b.Year, b.ReferenceOnly); err != nil {
				return nil, err
			}
			if _, _, ok := l.locate(b.ISBN); ok {
				return nil, fmt.Errorf("%w: %s", ErrDuplicateISBN, b.ISBN)
			}
			if err := s.Add(b); err != nil {
				return nil, err
			}
		}
	}
	for _, loan := range snap.Loans {
		if _, err := l.Lend(loan.ISBN, loan.Borrower); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// PersistWith registers a function that receives a snapshot after every
// change. It is called while the lock is held, so snapshots are saved in the
// order the changes happened. If it fails, the change is rolled back.
func (l *Library) PersistWith(persist func(Snapshot) error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.persist = persist
}

// Snapshot returns a copy of the current state.
func (l *Library) Snapshot() Snapshot {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.snapshot()
}

// snapshot expects l.mu to be held. Loans are ordered by shelf and book so
// that the output is deterministic.
func (l *Library) snapshot() Snapshot {
	snap := Snapshot{
		Shelves: make([]ShelfSnapshot, 0, len(l.shelves)),
		Loans:   make([]Loan, 0, len(l.loans)),
	}
	for _, s := range l.shelves {
		snap.Shelves = append(snap.Shelves, ShelfSnapshot{
			ID:       s.id,
			Capacity: s.capacity,
			Books:    append([]Book(nil), s.books...),
		})
		for _, b := range s.books {
			if loan, ok := l.loans[b.ISBN]; ok {
				snap.Loans = append(snap.Loans, loan)
			}
		}
	}
	return snap
}

// save expects l.mu to be held. On failure it calls rollback to undo the
// change that was just applied.
func (l *Library) save(rollback func()) error {
	if l.persist == nil {
		return nil
	}
	if err := l.persist(l.snapshot()); err != nil {
		rollback()
		return fmt.Errorf("persist: %w", err)
	}
	return nil
}
