package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"buecherei/internal/library"
)

const (
	isbnHobbit    = "978-3-608-93800-4"
	isbnCleanCode = "978-0-13-235088-4"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	lib := library.New()
	for _, id := range []string{"A", "B"} {
		if err := lib.AddShelf(id, 2); err != nil {
			t.Fatalf("AddShelf(%q) error = %v", id, err)
		}
	}

	hobbitISBN, err := library.ParseISBN(isbnHobbit)
	if err != nil {
		t.Fatalf("ParseISBN() error = %v", err)
	}
	hobbit, err := library.NewBook("Der Hobbit", "J. R. R. Tolkien", hobbitISBN, 1937)
	if err != nil {
		t.Fatalf("NewBook() error = %v", err)
	}
	cleanCodeISBN, err := library.ParseISBN(isbnCleanCode)
	if err != nil {
		t.Fatalf("ParseISBN() error = %v", err)
	}
	cleanCode, err := library.NewReferenceBook("Clean Code", "Robert C. Martin", cleanCodeISBN, 2008)
	if err != nil {
		t.Fatalf("NewReferenceBook() error = %v", err)
	}
	for _, b := range []library.Book{hobbit, cleanCode} {
		if _, err := lib.AddBook(b); err != nil {
			t.Fatalf("AddBook() error = %v", err)
		}
	}
	return NewHandler(lib)
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSearchBooks(t *testing.T) {
	h := newTestHandler(t)

	rec := do(t, h, http.MethodGet, "/books?q=hobbit", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var books []bookResponse
	if err := json.NewDecoder(rec.Body).Decode(&books); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(books) != 1 || books[0].Title != "Der Hobbit" || books[0].Shelf != "A" {
		t.Errorf("books = %+v, want only Der Hobbit on shelf A", books)
	}
}

func TestSearchBooksNoMatch(t *testing.T) {
	h := newTestHandler(t)

	rec := do(t, h, http.MethodGet, "/books?q=xyz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestLendAndReturn(t *testing.T) {
	h := newTestHandler(t)
	loanPath := "/books/" + isbnHobbit + "/loan"
	steps := []struct {
		name   string
		method string
		body   string
		want   int
	}{
		{"lend", http.MethodPost, `{"borrower":"Max"}`, http.StatusCreated},
		{"lend again", http.MethodPost, `{"borrower":"Max"}`, http.StatusConflict},
		{"return", http.MethodDelete, "", http.StatusNoContent},
		{"return again", http.MethodDelete, "", http.StatusConflict},
	}
	for _, s := range steps {
		rec := do(t, h, s.method, loanPath, s.body)
		if rec.Code != s.want {
			t.Fatalf("%s: status = %d, want %d (body %s)", s.name, rec.Code, s.want, rec.Body)
		}
		if s.want == http.StatusNoContent && rec.Body.Len() != 0 {
			t.Errorf("%s: body = %q, want empty", s.name, rec.Body)
		}
	}
}

func TestLendResponse(t *testing.T) {
	h := newTestHandler(t)

	rec := do(t, h, http.MethodPost, "/books/"+isbnHobbit+"/loan", `{"borrower":"Max"}`)
	var loan loanResponse
	if err := json.NewDecoder(rec.Body).Decode(&loan); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if loan != (loanResponse{ISBN: "9783608938004", Borrower: "Max"}) {
		t.Errorf("loan = %+v", loan)
	}
}

func TestAddBook(t *testing.T) {
	h := newTestHandler(t)

	rec := do(t, h, http.MethodPost, "/books",
		`{"title":"Momo","author":"Michael Ende","isbn":"978-3-522-20210-1","year":1973}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusCreated, rec.Body)
	}
	var b bookResponse
	if err := json.NewDecoder(rec.Body).Decode(&b); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := bookResponse{Title: "Momo", Author: "Michael Ende", ISBN: "9783522202101", Year: 1973, Shelf: "B"}
	if b != want {
		t.Errorf("book = %+v, want %+v", b, want)
	}

	rec = do(t, h, http.MethodGet, "/books?q=momo", "")
	if !strings.Contains(rec.Body.String(), "Momo") {
		t.Errorf("search after add: body = %s", rec.Body)
	}
}

func TestAddBookErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"malformed json", `{`, http.StatusBadRequest},
		{"invalid isbn", `{"title":"X","author":"Y","isbn":"abc","year":2000}`, http.StatusBadRequest},
		{"empty title", `{"title":"","author":"Y","isbn":"9780000000000","year":2000}`, http.StatusBadRequest},
		{"duplicate isbn", `{"title":"X","author":"Y","isbn":"` + isbnHobbit + `","year":2000}`, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(t)

			rec := do(t, h, http.MethodPost, "/books", tt.body)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestAddBookNoSpace(t *testing.T) {
	h := newTestHandler(t)

	// Shelves A and B hold 2 books each; the test library starts with 2.
	for i, want := range []int{http.StatusCreated, http.StatusCreated, http.StatusConflict} {
		body := fmt.Sprintf(`{"title":"Buch %d","author":"Y","isbn":"978000000000%d","year":2000}`, i, i)
		if rec := do(t, h, http.MethodPost, "/books", body); rec.Code != want {
			t.Fatalf("book %d: status = %d, want %d (body %s)", i, rec.Code, want, rec.Body)
		}
	}
}

func TestLendErrors(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{"reference only", "/books/" + isbnCleanCode + "/loan", `{"borrower":"Max"}`, http.StatusConflict},
		{"unknown isbn", "/books/9780000000000/loan", `{"borrower":"Max"}`, http.StatusNotFound},
		{"invalid isbn", "/books/abc/loan", `{"borrower":"Max"}`, http.StatusBadRequest},
		{"malformed json", "/books/" + isbnHobbit + "/loan", `{`, http.StatusBadRequest},
		{"empty borrower", "/books/" + isbnHobbit + "/loan", `{"borrower":""}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(t)

			rec := do(t, h, http.MethodPost, tt.path, tt.body)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.want, rec.Body)
			}
			var e errorResponse
			if err := json.NewDecoder(rec.Body).Decode(&e); err != nil || e.Error == "" {
				t.Errorf("error body = %+v, decode err = %v", e, err)
			}
		})
	}
}
