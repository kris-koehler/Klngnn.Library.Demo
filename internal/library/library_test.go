package library

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

const (
	isbnHobbit    ISBN = "9783608938004"
	isbnPanem     ISBN = "9783789132185"
	isbnCleanCode ISBN = "9780132350884"
)

// newTestLibrary returns a library with shelves A and B (2 slots each) and
// the three sample books.
func newTestLibrary(t *testing.T) *Library {
	t.Helper()
	lib := New()
	for _, id := range []string{"A", "B"} {
		if err := lib.AddShelf(id, 2); err != nil {
			t.Fatalf("AddShelf(%q) error = %v", id, err)
		}
	}

	cleanCode, err := NewReferenceBook("Clean Code", "Robert C. Martin", isbnCleanCode, 2008)
	if err != nil {
		t.Fatalf("NewReferenceBook() error = %v", err)
	}
	books := []Book{
		mustBook(t, "Der Hobbit", "J. R. R. Tolkien", isbnHobbit, 1937),
		mustBook(t, "Die Tribute von Panem", "Suzanne Collins", isbnPanem, 2008),
		cleanCode,
	}
	for _, b := range books {
		if _, err := lib.AddBook(b); err != nil {
			t.Fatalf("AddBook(%q) error = %v", b.Title, err)
		}
	}
	return lib
}

func TestAddBookFillsShelvesInOrder(t *testing.T) {
	lib := newTestLibrary(t)

	want := map[ISBN]string{isbnHobbit: "A", isbnPanem: "A", isbnCleanCode: "B"}
	for _, r := range lib.Search("") {
		if r.ShelfID != want[r.Book.ISBN] {
			t.Errorf("%q on shelf %q, want %q", r.Book.Title, r.ShelfID, want[r.Book.ISBN])
		}
	}
}

func TestAddBookNoSpace(t *testing.T) {
	lib := newTestLibrary(t)
	if _, err := lib.AddBook(mustBook(t, "Vierter", "Autor", "9780000000004", 2000)); err != nil {
		t.Fatalf("AddBook() 4th book error = %v", err)
	}

	_, err := lib.AddBook(mustBook(t, "Fünfter", "Autor", "9780000000005", 2000))
	if !errors.Is(err, ErrNoSpace) {
		t.Errorf("AddBook() 5th book error = %v, want %v", err, ErrNoSpace)
	}
}

func TestAddBookDuplicateISBN(t *testing.T) {
	lib := newTestLibrary(t)

	_, err := lib.AddBook(mustBook(t, "Kopie", "Autor", isbnHobbit, 2000))
	if !errors.Is(err, ErrDuplicateISBN) {
		t.Errorf("AddBook() error = %v, want %v", err, ErrDuplicateISBN)
	}
}

func TestAddShelfInvalid(t *testing.T) {
	lib := newTestLibrary(t)

	if err := lib.AddShelf("A", 2); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("AddShelf() duplicate id error = %v, want %v", err, ErrInvalidInput)
	}
	if err := lib.AddShelf("C", 0); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("AddShelf() capacity 0 error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestSearch(t *testing.T) {
	lib := newTestLibrary(t)
	tests := []struct {
		query string
		want  []ISBN
	}{
		{"hobbit", []ISBN{isbnHobbit}},
		{"TOLKIEN", []ISBN{isbnHobbit}},
		{"978-0-13", []ISBN{isbnCleanCode}},
		{"", []ISBN{isbnHobbit, isbnPanem, isbnCleanCode}},
		{"xyz", []ISBN{}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := lib.Search(tt.query)
			if got == nil {
				t.Fatal("Search() = nil, want non-nil slice")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Search(%q) returned %d results, want %d", tt.query, len(got), len(tt.want))
			}
			for i, r := range got {
				if r.Book.ISBN != tt.want[i] {
					t.Errorf("Search(%q)[%d] = %s, want %s", tt.query, i, r.Book.ISBN, tt.want[i])
				}
			}
		})
	}
}

func TestLend(t *testing.T) {
	lib := newTestLibrary(t)

	loan, err := lib.Lend(isbnHobbit, "Max")
	if err != nil {
		t.Fatalf("Lend() error = %v", err)
	}
	if loan != (Loan{ISBN: isbnHobbit, Borrower: "Max"}) {
		t.Errorf("Lend() = %+v", loan)
	}
	if !isLent(t, lib, isbnHobbit) {
		t.Error("Search() shows book as not lent after Lend()")
	}
}

func TestLendErrors(t *testing.T) {
	tests := []struct {
		name     string
		isbn     ISBN
		borrower string
		wantErr  error
	}{
		{"already lent", isbnHobbit, "Erika", ErrAlreadyLent},
		{"reference only", isbnCleanCode, "Erika", ErrReferenceOnly},
		{"unknown isbn", "9780000000000", "Erika", ErrBookNotFound},
		{"blank borrower", isbnPanem, "  ", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := newTestLibrary(t)
			if _, err := lib.Lend(isbnHobbit, "Max"); err != nil {
				t.Fatalf("Lend() setup error = %v", err)
			}

			if _, err := lib.Lend(tt.isbn, tt.borrower); !errors.Is(err, tt.wantErr) {
				t.Errorf("Lend() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestReturn(t *testing.T) {
	lib := newTestLibrary(t)
	if _, err := lib.Lend(isbnHobbit, "Max"); err != nil {
		t.Fatalf("Lend() error = %v", err)
	}

	if err := lib.Return(isbnHobbit); err != nil {
		t.Fatalf("Return() error = %v", err)
	}
	if isLent(t, lib, isbnHobbit) {
		t.Error("Search() shows book as lent after Return()")
	}
	if err := lib.Return(isbnHobbit); !errors.Is(err, ErrNotLent) {
		t.Errorf("Return() twice error = %v, want %v", err, ErrNotLent)
	}
}

func TestReturnUnknownISBN(t *testing.T) {
	lib := newTestLibrary(t)

	if err := lib.Return("9780000000000"); !errors.Is(err, ErrBookNotFound) {
		t.Errorf("Return() error = %v, want %v", err, ErrBookNotFound)
	}
}

func TestLentBookKeepsShelfSlot(t *testing.T) {
	lib := newTestLibrary(t)
	if _, err := lib.AddBook(mustBook(t, "Vierter", "Autor", "9780000000004", 2000)); err != nil {
		t.Fatalf("AddBook() error = %v", err)
	}
	if _, err := lib.Lend(isbnHobbit, "Max"); err != nil {
		t.Fatalf("Lend() error = %v", err)
	}

	_, err := lib.AddBook(mustBook(t, "Fünfter", "Autor", "9780000000005", 2000))
	if !errors.Is(err, ErrNoSpace) {
		t.Errorf("AddBook() error = %v, want %v", err, ErrNoSpace)
	}
}

func TestLendConcurrent(t *testing.T) {
	lib := newTestLibrary(t)
	const n = 50

	var wg sync.WaitGroup
	var succeeded, alreadyLent atomic.Int32
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := lib.Lend(isbnHobbit, "Max")
			switch {
			case err == nil:
				succeeded.Add(1)
			case errors.Is(err, ErrAlreadyLent):
				alreadyLent.Add(1)
			default:
				t.Errorf("Lend() unexpected error = %v", err)
			}
		}()
	}
	wg.Wait()

	if got := succeeded.Load(); got != 1 {
		t.Errorf("successful loans = %d, want 1", got)
	}
	if got := alreadyLent.Load(); got != n-1 {
		t.Errorf("ErrAlreadyLent = %d, want %d", got, n-1)
	}
}

func TestShelves(t *testing.T) {
	if got := New().Shelves(); got == nil || len(got) != 0 {
		t.Errorf("Shelves() on empty library = %#v, want empty non-nil slice", got)
	}

	lib := newTestLibrary(t)
	if _, err := lib.Lend(isbnHobbit, "Max"); err != nil {
		t.Fatalf("Lend() error = %v", err)
	}

	want := []ShelfInfo{{ID: "A", Capacity: 2, Used: 2, Lent: 1}, {ID: "B", Capacity: 2, Used: 1, Lent: 0}}
	if got := lib.Shelves(); !reflect.DeepEqual(got, want) {
		t.Errorf("Shelves() = %+v, want %+v", got, want)
	}
}

func isLent(t *testing.T, lib *Library, isbn ISBN) bool {
	t.Helper()
	for _, r := range lib.Search(string(isbn)) {
		if r.Book.ISBN == isbn {
			return r.Lent
		}
	}
	t.Fatalf("book %s not found", isbn)
	return false
}
