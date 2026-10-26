package library

import "errors"

var (
	ErrInvalidISBN   = errors.New("invalid isbn")
	ErrInvalidInput  = errors.New("invalid input")
	ErrShelfFull     = errors.New("shelf is full")
	ErrNoSpace       = errors.New("no shelf has free space")
	ErrDuplicateISBN = errors.New("book with this isbn already exists")
	ErrBookNotFound  = errors.New("book not found")
	ErrReferenceOnly = errors.New("reference-only book cannot be lent")
	ErrAlreadyLent   = errors.New("book is already lent")
	ErrNotLent       = errors.New("book is not lent")
)
