// Package api exposes the library via a JSON HTTP API.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"

	"buecherei/internal/auth"
	"buecherei/internal/library"
)

const maxBodyBytes = 1 << 20

var (
	errUnauthorized = errors.New("missing or invalid bearer token")
	errForbidden    = errors.New("not allowed for this role")
	errEmptyBody    = errors.New("empty body")
)

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

type meResponse struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	lib    *library.Library
	tokens *auth.Tokens
}

// authedHandler is a handler that runs for an authenticated principal.
type authedHandler func(w http.ResponseWriter, r *http.Request, p auth.Principal)

// NewHandler returns an http.Handler serving the library API. Searching is
// public; all other routes require a bearer token with a suitable role.
func NewHandler(lib *library.Library, tokens *auth.Tokens) http.Handler {
	h := &handler{lib: lib, tokens: tokens}
	anyRole := []auth.Role{auth.RoleCustomer, auth.RoleStaff}
	staff := []auth.Role{auth.RoleStaff}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /books", h.searchBooks)
	mux.HandleFunc("GET /me", h.require(anyRole, h.me))
	mux.HandleFunc("POST /books/{isbn}/loan", h.require(anyRole, h.lendBook))
	mux.HandleFunc("DELETE /books/{isbn}/loan", h.require(staff, h.returnBook))
	mux.HandleFunc("POST /books", h.require(staff, h.addBook))
	mux.HandleFunc("GET /shelves", h.require(staff, h.listShelves))
	return mux
}

// require authenticates the request and checks that the principal has one of
// the given roles before calling next.
func (h *handler) require(roles []auth.Role, next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := h.tokens.Authenticate(bearerToken(r))
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		if !slices.Contains(roles, p.Role) {
			writeError(w, errForbidden)
			return
		}
		next(w, r, p)
	}
}

func bearerToken(r *http.Request) string {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func (h *handler) me(w http.ResponseWriter, r *http.Request, p auth.Principal) {
	writeJSON(w, http.StatusOK, meResponse{Name: p.Name, Role: string(p.Role)})
}

func (h *handler) listShelves(w http.ResponseWriter, r *http.Request, _ auth.Principal) {
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

func (h *handler) addBook(w http.ResponseWriter, r *http.Request, _ auth.Principal) {
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

// lendBook lets staff lend to anyone named in the body. Customers always
// borrow for themselves; the body is optional for them, and naming someone
// else is forbidden.
func (h *handler) lendBook(w http.ResponseWriter, r *http.Request, p auth.Principal) {
	isbn, err := library.ParseISBN(r.PathValue("isbn"))
	if err != nil {
		writeError(w, err)
		return
	}

	var req loanRequest
	err = decodeJSON(w, r, &req)
	isCustomer := p.Role == auth.RoleCustomer
	if err != nil && !(isCustomer && errors.Is(err, errEmptyBody)) {
		writeError(w, err)
		return
	}

	borrower := req.Borrower
	if isCustomer {
		if strings.TrimSpace(borrower) != "" && strings.TrimSpace(borrower) != p.Name {
			writeError(w, fmt.Errorf("%w: customers can only borrow for themselves", errForbidden))
			return
		}
		borrower = p.Name
	}

	loan, err := h.lib.Lend(isbn, borrower)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, loanResponse{ISBN: string(loan.ISBN), Borrower: loan.Borrower})
}

func (h *handler) returnBook(w http.ResponseWriter, r *http.Request, _ auth.Principal) {
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

// decodeJSON reads a size-limited JSON body into v. An empty body yields an
// error that matches both library.ErrInvalidInput and errEmptyBody.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(v)
	switch {
	case errors.Is(err, io.EOF):
		return fmt.Errorf("%w: %w", library.ErrInvalidInput, errEmptyBody)
	case err != nil:
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
	switch status {
	case http.StatusUnauthorized:
		w.Header().Set("WWW-Authenticate", "Bearer")
	case http.StatusInternalServerError:
		log.Printf("internal error: %v", err)
		msg = "internal error"
	}
	writeJSON(w, status, errorResponse{Error: msg})
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, errUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, errForbidden):
		return http.StatusForbidden
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
