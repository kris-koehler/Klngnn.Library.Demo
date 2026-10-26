// Package auth maps static bearer tokens to principals with a role.
package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Role decides which API operations a principal may use.
type Role string

const (
	RoleCustomer Role = "customer"
	RoleStaff    Role = "staff"
)

// minTokenLength rejects trivially guessable tokens.
const minTokenLength = 16

// ErrInvalidConfig is returned for an unusable token configuration.
var ErrInvalidConfig = errors.New("invalid token configuration")

// Principal is an authenticated caller.
type Principal struct {
	Name string
	Role Role
}

// Entry is one configured token.
type Entry struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	Role  Role   `json:"role"`
}

type tokenEntry struct {
	hash      [sha256.Size]byte
	principal Principal
}

// Tokens authenticates bearer tokens. It is safe for concurrent use because
// it is never modified after creation.
type Tokens struct {
	entries []tokenEntry
}

// New validates the entries and returns a token set. Only hashes of the
// tokens are kept in memory.
func New(entries []Entry) (*Tokens, error) {
	t := &Tokens{entries: make([]tokenEntry, 0, len(entries))}
	seen := make(map[[sha256.Size]byte]bool, len(entries))
	for i, e := range entries {
		if len(e.Token) < minTokenLength {
			return nil, fmt.Errorf("%w: entry %d: token must have at least %d characters", ErrInvalidConfig, i, minTokenLength)
		}
		if strings.TrimSpace(e.Name) == "" {
			return nil, fmt.Errorf("%w: entry %d: name must not be empty", ErrInvalidConfig, i)
		}
		if e.Role != RoleCustomer && e.Role != RoleStaff {
			return nil, fmt.Errorf("%w: entry %d: unknown role %q", ErrInvalidConfig, i, e.Role)
		}
		hash := sha256.Sum256([]byte(e.Token))
		if seen[hash] {
			return nil, fmt.Errorf("%w: entry %d: duplicate token", ErrInvalidConfig, i)
		}
		seen[hash] = true
		t.entries = append(t.entries, tokenEntry{hash: hash, principal: Principal{Name: strings.TrimSpace(e.Name), Role: e.Role}})
	}
	return t, nil
}

// Load reads the token configuration from a JSON file containing a list of
// entries. If the file does not exist, the error matches os.ErrNotExist.
func Load(path string) (*Tokens, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %v", ErrInvalidConfig, path, err)
	}
	return New(entries)
}

// Authenticate returns the principal for the token. Hashing first gives
// fixed-length values, and all entries are compared in constant time, so the
// timing reveals neither which token matched nor how much of it.
func (t *Tokens) Authenticate(token string) (Principal, bool) {
	if token == "" {
		return Principal{}, false
	}
	hash := sha256.Sum256([]byte(token))
	var found Principal
	ok := false
	for _, e := range t.entries {
		if subtle.ConstantTimeCompare(hash[:], e.hash[:]) == 1 {
			found, ok = e.principal, true
		}
	}
	return found, ok
}
