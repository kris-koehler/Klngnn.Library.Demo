package storage

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"buecherei/internal/library"
)

func TestFileStoreRoundTrip(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "data.json"))
	snap := library.Snapshot{
		Shelves: []library.ShelfSnapshot{
			{ID: "A", Capacity: 2, Books: []library.Book{
				{Title: "Der Hobbit", Author: "J. R. R. Tolkien", ISBN: "9783608938004", Year: 1937},
				{Title: "Clean Code", Author: "Robert C. Martin", ISBN: "9780132350884", Year: 2008, ReferenceOnly: true},
			}},
		},
		Loans: []library.Loan{{ISBN: "9783608938004", Borrower: "Max"}},
	}

	if err := store.Save(snap); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got, snap) {
		t.Errorf("Load() = %+v, want %+v", got, snap)
	}
}

func TestFileStoreSaveOverwritesWithoutLeftovers(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "data.json"))

	for range 2 {
		if err := store.Save(library.Snapshot{}); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory contains %d files, want only data.json", len(entries))
	}
}

func TestFileStoreLoadMissing(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))

	if _, err := store.Load(); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Load() error = %v, want os.ErrNotExist", err)
	}
}

func TestFileStoreLoadInvalid(t *testing.T) {
	tests := map[string]string{
		"malformed json":      `{`,
		"unsupported version": `{"version":99}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "data.json")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := NewFileStore(path).Load(); err == nil {
				t.Error("Load() error = nil, want error")
			}
		})
	}
}
