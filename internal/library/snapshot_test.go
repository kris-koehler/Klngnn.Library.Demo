package library

import (
	"errors"
	"reflect"
	"testing"
)

func TestSnapshotRoundTrip(t *testing.T) {
	lib := newTestLibrary(t)
	if _, err := lib.Lend(isbnHobbit, "Max"); err != nil {
		t.Fatalf("Lend() error = %v", err)
	}
	snap := lib.Snapshot()

	restored, err := NewFromSnapshot(snap)
	if err != nil {
		t.Fatalf("NewFromSnapshot() error = %v", err)
	}
	if got := restored.Snapshot(); !reflect.DeepEqual(got, snap) {
		t.Errorf("restored snapshot = %+v, want %+v", got, snap)
	}
	if !isLent(t, restored, isbnHobbit) {
		t.Error("restored library lost the loan")
	}
}

func TestNewFromSnapshotInvalid(t *testing.T) {
	hobbit := mustBook(t, "Der Hobbit", "J. R. R. Tolkien", isbnHobbit, 1937)
	tests := []struct {
		name    string
		snap    Snapshot
		wantErr error
	}{
		{"duplicate shelf", Snapshot{Shelves: []ShelfSnapshot{{ID: "A", Capacity: 1}, {ID: "A", Capacity: 1}}}, ErrInvalidInput},
		{"overfull shelf", Snapshot{Shelves: []ShelfSnapshot{{ID: "A", Capacity: 1, Books: []Book{hobbit, {Title: "X", Author: "Y", ISBN: isbnPanem, Year: 1}}}}}, ErrShelfFull},
		{"duplicate isbn", Snapshot{Shelves: []ShelfSnapshot{{ID: "A", Capacity: 2, Books: []Book{hobbit, hobbit}}}}, ErrDuplicateISBN},
		{"invalid book", Snapshot{Shelves: []ShelfSnapshot{{ID: "A", Capacity: 1, Books: []Book{{Title: "X", Author: "Y", ISBN: isbnHobbit}}}}}, ErrInvalidInput},
		{"loan of unknown book", Snapshot{Loans: []Loan{{ISBN: isbnHobbit, Borrower: "Max"}}}, ErrBookNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewFromSnapshot(tt.snap); !errors.Is(err, tt.wantErr) {
				t.Errorf("NewFromSnapshot() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPersistCalledOnChange(t *testing.T) {
	lib := newTestLibrary(t)
	var saved []Snapshot
	lib.PersistWith(func(s Snapshot) error {
		saved = append(saved, s)
		return nil
	})

	if _, err := lib.Lend(isbnHobbit, "Max"); err != nil {
		t.Fatalf("Lend() error = %v", err)
	}
	if err := lib.Return(isbnHobbit); err != nil {
		t.Fatalf("Return() error = %v", err)
	}
	if _, err := lib.Lend(isbnCleanCode, "Max"); err == nil {
		t.Fatal("Lend() of reference book succeeded")
	}

	if len(saved) != 2 {
		t.Fatalf("persist called %d times, want 2 (failed operations must not persist)", len(saved))
	}
	if len(saved[0].Loans) != 1 || len(saved[1].Loans) != 0 {
		t.Errorf("saved loans = %v, %v", saved[0].Loans, saved[1].Loans)
	}
}

func TestPersistFailureRollsBack(t *testing.T) {
	errDisk := errors.New("disk full")
	lib := newTestLibrary(t)
	if _, err := lib.Lend(isbnPanem, "Erika"); err != nil {
		t.Fatalf("Lend() error = %v", err)
	}
	before := lib.Snapshot()
	lib.PersistWith(func(Snapshot) error { return errDisk })

	if _, err := lib.Lend(isbnHobbit, "Max"); !errors.Is(err, errDisk) {
		t.Errorf("Lend() error = %v, want %v", err, errDisk)
	}
	if err := lib.Return(isbnPanem); !errors.Is(err, errDisk) {
		t.Errorf("Return() error = %v, want %v", err, errDisk)
	}
	if _, err := lib.AddBook(mustBook(t, "Vierter", "Autor", "9780000000004", 2000)); !errors.Is(err, errDisk) {
		t.Errorf("AddBook() error = %v, want %v", err, errDisk)
	}
	if err := lib.AddShelf("C", 1); !errors.Is(err, errDisk) {
		t.Errorf("AddShelf() error = %v, want %v", err, errDisk)
	}

	if got := lib.Snapshot(); !reflect.DeepEqual(got, before) {
		t.Errorf("state after failed saves = %+v, want %+v", got, before)
	}
}
