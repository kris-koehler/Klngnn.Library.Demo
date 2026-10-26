package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	customerToken = "customer-token-0001"
	staffToken    = "staff-token-000001"
)

func newTestTokens(t *testing.T) *Tokens {
	t.Helper()
	tokens, err := New([]Entry{
		{Token: customerToken, Name: "Max", Role: RoleCustomer},
		{Token: staffToken, Name: "Theke", Role: RoleStaff},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return tokens
}

func TestAuthenticate(t *testing.T) {
	tokens := newTestTokens(t)
	tests := []struct {
		name   string
		token  string
		want   Principal
		wantOK bool
	}{
		{"customer", customerToken, Principal{Name: "Max", Role: RoleCustomer}, true},
		{"staff", staffToken, Principal{Name: "Theke", Role: RoleStaff}, true},
		{"unknown", "unknown-token-00000", Principal{}, false},
		{"prefix of valid token", customerToken[:10], Principal{}, false},
		{"empty", "", Principal{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tokens.Authenticate(tt.token)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("Authenticate(%q) = %+v, %v, want %+v, %v", tt.token, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNewInvalid(t *testing.T) {
	tests := []struct {
		name    string
		entries []Entry
	}{
		{"short token", []Entry{{Token: "short", Name: "Max", Role: RoleCustomer}}},
		{"empty name", []Entry{{Token: customerToken, Name: " ", Role: RoleCustomer}}},
		{"unknown role", []Entry{{Token: customerToken, Name: "Max", Role: "admin"}}},
		{"duplicate token", []Entry{
			{Token: customerToken, Name: "Max", Role: RoleCustomer},
			{Token: customerToken, Name: "Erika", Role: RoleCustomer},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.entries); !errors.Is(err, ErrInvalidConfig) {
				t.Errorf("New() error = %v, want %v", err, ErrInvalidConfig)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	content := `[{"token":"` + staffToken + `","name":"Theke","role":"staff"}]`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	tokens, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if p, ok := tokens.Authenticate(staffToken); !ok || p.Role != RoleStaff {
		t.Errorf("Authenticate() = %+v, %v", p, ok)
	}

	if _, err := Load(filepath.Join(dir, "missing.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Load() missing file error = %v, want os.ErrNotExist", err)
	}
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("Load() malformed error = %v, want %v", err, ErrInvalidConfig)
	}
}

func TestExampleFileIsValid(t *testing.T) {
	if _, err := Load(filepath.Join("..", "..", "tokens.example.json")); err != nil {
		t.Errorf("tokens.example.json: %v", err)
	}
}
