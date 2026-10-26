// Package storage persists library snapshots as a JSON file.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"buecherei/internal/library"
)

const formatVersion = 1

type fileData struct {
	Version int         `json:"version"`
	Shelves []shelfData `json:"shelves"`
	Loans   []loanData  `json:"loans"`
}

type shelfData struct {
	ID       string     `json:"id"`
	Capacity int        `json:"capacity"`
	Books    []bookData `json:"books"`
}

type bookData struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn"`
	Year          int    `json:"year"`
	ReferenceOnly bool   `json:"referenceOnly"`
}

type loanData struct {
	ISBN     string `json:"isbn"`
	Borrower string `json:"borrower"`
}

// FileStore reads and writes library snapshots from a single JSON file.
type FileStore struct {
	path string
}

// NewFileStore returns a store for the file at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Load reads the snapshot from the file. If the file does not exist, the
// returned error matches os.ErrNotExist.
func (s *FileStore) Load() (library.Snapshot, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return library.Snapshot{}, err
	}
	var data fileData
	if err := json.Unmarshal(raw, &data); err != nil {
		return library.Snapshot{}, fmt.Errorf("decode %s: %w", s.path, err)
	}
	if data.Version != formatVersion {
		return library.Snapshot{}, fmt.Errorf("%s: unsupported format version %d", s.path, data.Version)
	}
	return toSnapshot(data), nil
}

// Save writes the snapshot atomically: it writes a temporary file in the same
// directory and renames it, so a crash never leaves a half-written file.
func (s *FileStore) Save(snap library.Snapshot) error {
	raw, err := json.MarshalIndent(fromSnapshot(snap), "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}

func fromSnapshot(snap library.Snapshot) fileData {
	data := fileData{
		Version: formatVersion,
		Shelves: make([]shelfData, 0, len(snap.Shelves)),
		Loans:   make([]loanData, 0, len(snap.Loans)),
	}
	for _, sh := range snap.Shelves {
		books := make([]bookData, 0, len(sh.Books))
		for _, b := range sh.Books {
			books = append(books, bookData{
				Title:         b.Title,
				Author:        b.Author,
				ISBN:          string(b.ISBN),
				Year:          b.Year,
				ReferenceOnly: b.ReferenceOnly,
			})
		}
		data.Shelves = append(data.Shelves, shelfData{ID: sh.ID, Capacity: sh.Capacity, Books: books})
	}
	for _, l := range snap.Loans {
		data.Loans = append(data.Loans, loanData{ISBN: string(l.ISBN), Borrower: l.Borrower})
	}
	return data
}

// toSnapshot converts without validation; library.NewFromSnapshot validates.
func toSnapshot(data fileData) library.Snapshot {
	var snap library.Snapshot
	for _, sh := range data.Shelves {
		var books []library.Book
		for _, b := range sh.Books {
			books = append(books, library.Book{
				Title:         b.Title,
				Author:        b.Author,
				ISBN:          library.ISBN(b.ISBN),
				Year:          b.Year,
				ReferenceOnly: b.ReferenceOnly,
			})
		}
		snap.Shelves = append(snap.Shelves, library.ShelfSnapshot{ID: sh.ID, Capacity: sh.Capacity, Books: books})
	}
	for _, l := range data.Loans {
		snap.Loans = append(snap.Loans, library.Loan{ISBN: library.ISBN(l.ISBN), Borrower: l.Borrower})
	}
	return snap
}
