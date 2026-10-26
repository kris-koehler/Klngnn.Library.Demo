// Package api exposes the library via a JSON HTTP API.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"buecherei/internal/library"
)

const maxBodyBytes = 1 << 20

type bookResponse struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn"`
	Year          int    `json:"year"`
	ReferenceOnly bool   `json:"referenceOnly"`
	Shelf         string `json:"shelf"`
	Lent          bool   `json:"lent"`
}

type bookRequest struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn"`
	Year          int    `json:"year"`
	ReferenceOnly bool   `json:"referenceOnly"`
}

type loanRequest struct {
	Borrower string `json:"borrower"`
}

type loanResponse struct {
	ISBN     string `json:"isbn"`
	Borrower string `json:"borrower"`
}

type shelfResponse struct {
	ID       string `json:"id"`
	Capacity int    `json:"capacity"`
	Used     int    `json:"used"`
	Free     int    `json:"free"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	lib *library.Library
}

// NewHandler returns an http.Handler serving the library API.
func NewHandler(lib *library.Library) http.Handler {
	h := &handler{lib: lib}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /books", h.searchBooks)
	mux.HandleFunc("POST /books", h.addBook)
	mux.HandleFunc("POST /books/{isbn}/loan", h.lendBook)
	mux.HandleFunc("DELETE /books/{isbn}/loan", h.returnBook)
	mux.HandleFunc("GET /shelves", h.listShelves)
	return mux
}

func (h *handler) listShelves(w http.ResponseWriter, r *http.Request) {
	infos := h.lib.Shelves()

	shelves := make([]shelfResponse, 0, len(infos))
	for _, s := range infos {
		shelves = append(shelves, shelfResponse{ID: s.ID, Capacity: s.Capacity, Used: s.Used, Free: s.Capacity - s.Used})
	}
	writeJSON(w, http.StatusOK, shelves)
}

func (h *handler) searchBooks(w http.ResponseWriter, r *http.Request) {
	results := h.lib.Search(r.URL.Query().Get("q"))

	books := make([]bookResponse, 0, len(results))
	for _, res := range results {
		books = append(books, toBookResponse(res.Book, res.ShelfID, res.Lent))
	}
	writeJSON(w, http.StatusOK, books)
}

func (h *handler) addBook(w http.ResponseWriter, r *http.Request) {
	var req bookRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	isbn, err := library.ParseISBN(req.ISBN)
	if err != nil {
		writeError(w, err)
		return
	}

	newBook := library.NewBook
	if req.ReferenceOnly {
		newBook = library.NewReferenceBook
	}
	b, err := newBook(req.Title, req.Author, isbn, req.Year)
	if err != nil {
		writeError(w, err)
		return
	}

	shelfID, err := h.lib.AddBook(b)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toBookResponse(b, shelfID, false))
}

func (h *handler) lendBook(w http.ResponseWriter, r *http.Request) {
	isbn, err := library.ParseISBN(r.PathValue("isbn"))
	if err != nil {
		writeError(w, err)
		return
	}

	var req loanRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}

	loan, err := h.lib.Lend(isbn, req.Borrower)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, loanResponse{ISBN: string(loan.ISBN), Borrower: loan.Borrower})
}

func (h *handler) returnBook(w http.ResponseWriter, r *http.Request) {
	isbn, err := library.ParseISBN(r.PathValue("isbn"))
	if err != nil {
		writeError(w, err)
		return
	}

	if err := h.lib.Return(isbn); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toBookResponse(b library.Book, shelfID string, lent bool) bookResponse {
	return bookResponse{
		Title:         b.Title,
		Author:        b.Author,
		ISBN:          string(b.ISBN),
		Year:          b.Year,
		ReferenceOnly: b.ReferenceOnly,
		Shelf:         shelfID,
		Lent:          lent,
	}
}

// decodeJSON reads a size-limited JSON body into v.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("%w: malformed JSON body", library.ErrInvalidInput)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := statusFor(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		log.Printf("internal error: %v", err)
		msg = "internal error"
	}
	writeJSON(w, status, errorResponse{Error: msg})
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, library.ErrInvalidISBN),
		errors.Is(err, library.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, library.ErrBookNotFound):
		return http.StatusNotFound
	case errors.Is(err, library.ErrAlreadyLent),
		errors.Is(err, library.ErrNotLent),
		errors.Is(err, library.ErrReferenceOnly),
		errors.Is(err, library.ErrDuplicateISBN),
		errors.Is(err, library.ErrNoSpace):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
