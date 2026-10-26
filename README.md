# Library (Demo)
#### Do not even think about using this in production yet!

## English

A dependency-free Go backend that manages a library with capacity-limited shelves and offers searching, lending, returning and adding books as well as shelf occupancy via a JSON HTTP API and a minimal browser frontend; run it with `cp tokens.example.json tokens.json && go run ./cmd/server`, open http://localhost:8080 and test it with `go test ./...` (or `go test -race ./...` where cgo and a C compiler are available – they were not on the development machine, so the concurrency test was run with `-count=200` instead).

The domain (`internal/library`) knows nothing about HTTP or files, loan state is kept centrally in the library rather than in the book, and lent books keep their shelf slot so a return can never fail because a shelf is full.

Reference-only is a property of a book instead of a subclass, errors are sentinel errors that the API maps to 400/401/403/404/409, and a mutex keeps lending atomic under concurrent requests.

The API separates self-service from administration by role rather than by separate servers: searching is public, customers use a bearer token to borrow only for themselves (the name comes from the token, not the request), and returning, adding books and viewing shelves are reserved for staff; tokens are configured statically in `tokens.json` and compared as SHA-256 hashes in constant time.

After every change the state is written atomically to `buecherei.json` while the lock is still held (temp file + rename); if saving fails the change is rolled back, and if the file is missing on start, sample data is created.

ISBNs are only validated for format because two of the three sample ISBNs have an invalid check digit; "Clean Code" is marked as reference-only for demonstration.

New books go to the chosen shelf or, if none is given, to the first one with free space; the shelf view shows how many of the books on each shelf are currently lent.

Next up are a role hierarchy with an `admin` role, shelf management, multiple copies per ISBN and real sign-in with accounts, sessions and HTTPS (see roadmap), followed by check-digit validation, member management with due dates, a database instead of the JSON file and further endpoints (single lookup, own loans).

## Deutsch

Umgesetzt ist ein Go-Backend ohne externe Abhängigkeiten, das eine Bücherei mit Regalen begrenzter Kapazität verwaltet und Suchen, Ausleihen, Zurückgeben und Anlegen von Büchern sowie die Regalbelegung über eine JSON-HTTP-API und ein schmales Browser-Frontend anbietet; Start mit `cp tokens.example.json tokens.json && go run ./cmd/server`, danach http://localhost:8080 öffnen, Tests mit `go test ./...` (bzw. `go test -race ./...`, sofern cgo und ein C-Compiler verfügbar sind – auf der Entwicklungsmaschine war das nicht der Fall, der Nebenläufigkeitstest lief stattdessen mit `-count=200`).

Die Domäne (`internal/library`) kennt weder HTTP noch Dateien, den Ausleihstatus verwaltet zentral die Bücherei statt das Buch, und ausgeliehene Bücher behalten ihren Regalplatz, damit eine Rückgabe nie an einem vollen Regal scheitert.

Präsenzbestand ist eine Eigenschaft des Buchs statt einer Unterklasse, Fehler sind Sentinel-Errors, die die API auf 400/401/403/404/409 abbildet, und ein Mutex macht das Ausleihen bei parallelen Requests atomar.

Die API trennt Self-Service und Verwaltung über Rollen statt über getrennte Server: Suchen ist öffentlich, Kundinnen und Kunden leihen per Bearer-Token nur für sich selbst aus (der Name kommt aus dem Token, nicht aus dem Request), Rückgabe, Anlegen von Büchern und Regalansicht sind dem Personal vorbehalten; die Tokens stehen statisch in `tokens.json` und werden als SHA-256-Hash in konstanter Zeit verglichen.

Der Zustand wird nach jeder Änderung noch unter dem Lock atomar in `buecherei.json` geschrieben (Temp-Datei + Rename); schlägt das Speichern fehl, wird die Änderung zurückgerollt, und fehlt die Datei beim Start, werden Beispieldaten angelegt.

Die ISBN wird nur auf ihr Format geprüft, weil zwei der drei Beispiel-ISBNs eine ungültige Prüfziffer haben; „Clean Code“ ist zur Demonstration als Präsenzbestand markiert.

Neue Bücher landen im gewählten Regal oder, ohne Angabe, im ersten mit freiem Platz; die Regalansicht zeigt neben der Belegung, wie viele der Bücher gerade ausgeliehen sind.

Als Nächstes stehen eine Rollenhierarchie mit einer `admin`-Rolle, Regalverwaltung, mehrere Exemplare pro ISBN und eine echte Anmeldung mit Konten, Sessions und HTTPS an (siehe Roadmap), danach Prüfziffer-Validierung, Personenverwaltung mit Leihfristen, eine Datenbank statt der JSON-Datei und weitere Endpunkte (Einzelabruf, eigene Ausleihen).

## Server

```bash
cp tokens.example.json tokens.json                 # demo tokens, see below
go run ./cmd/server                                # :8080, data in ./buecherei.json
go run ./cmd/server -addr :9000 -data /tmp/b.json -tokens /etc/buecherei/tokens.json
```

Delete the data file to start over with the sample data.

`tokens.example.json` contains demo tokens that are public in this repository – use your own `tokens.json` (ignored by git) for anything beyond a demo. Each entry has a `token` (at least 16 characters), a `name` and a `role` (`customer` or `staff`):

| Token | Name | Role |
|---|---|---|
| `demo-kunde-max-7f3a9c` | Max | customer |
| `demo-kunde-erika-2b8e41` | Erika | customer |
| `demo-personal-5d1c07e9` | Theke | staff |

## API

Authenticated requests send `Authorization: Bearer <token>`.

| Method | Path | Access | Body | Success |
|---|---|---|---|---|
| GET | `/books?q=…` | public | – | 200, list of books |
| GET | `/me` | customer, staff | – | 200, `{"name":"…","role":"…"}` |
| POST | `/books/{isbn}/loan` | customer, staff | staff: `{"borrower":"…"}`; customer: optional, own name only | 201, loan |
| DELETE | `/books/{isbn}/loan` | staff | – | 204 |
| POST | `/books` | staff | `{"title":"…","author":"…","isbn":"…","year":2000,"referenceOnly":false,"shelf":"B"}` (`shelf` optional) | 201, book incl. shelf |
| GET | `/shelves` | staff | – | 200, `[{"id":"A","capacity":2,"used":2,"lent":1,"free":0}]` (lent books keep their slot, so they count as used) |

Errors are returned as `{"error":"…"}` with status 400 (invalid input or unknown shelf), 401 (missing or unknown token), 403 (role not allowed, or a customer borrowing for someone else), 404 (unknown book), 409 (conflict, e.g. already lent, reference-only, duplicate ISBN, chosen shelf full or no free shelf space) or 500 (e.g. data file not writable).

```bash
KUNDE='Authorization: Bearer demo-kunde-max-7f3a9c'
PERSONAL='Authorization: Bearer demo-personal-5d1c07e9'

curl -s 'localhost:8080/books?q=hobbit'
curl -si -X POST -H "$KUNDE" localhost:8080/books/978-3-608-93800-4/loan      # 201, borrower Max
curl -si -X DELETE -H "$KUNDE" localhost:8080/books/978-3-608-93800-4/loan    # 403
curl -si -X DELETE -H "$PERSONAL" localhost:8080/books/978-3-608-93800-4/loan # 204
curl -si -X POST -H "$PERSONAL" -H 'Content-Type: application/json' \
  -d '{"title":"Momo","author":"Michael Ende","isbn":"978-3-522-20210-1","year":1973}' \
  localhost:8080/books
```

## Roadmap

### Role hierarchy

Today there are two roles: `customer` (self-service) and `staff` (front desk). A third, higher role `admin` is planned for managing the library itself. Roles become hierarchical, so each role includes the rights of the one below it:

| Role | Rights |
|---|---|
| `customer` | search, borrow for themselves, see own loans |
| `staff` | everything of `customer`, plus lending for anyone, returns, adding books, shelf view |
| `admin` | everything of `staff`, plus shelf management and (with accounts) managing users and roles |

Routes then declare a minimum role (`require(auth.RoleStaff, …)`) instead of a list of allowed roles, so a new role does not require touching every route. The role is introduced together with the first admin-only feature, shelf management; before that it would be identical to `staff`.

### Shelf management (admin)

Today shelves can only be created in code (`cmd/server/seed.go`).

| Operation | Rule |
|---|---|
| Add shelf (`POST /shelves`) | unique ID, capacity > 0 |
| Resize shelf | growing is always allowed; shrinking only down to the number of books on it, lent books included (their slot is reserved), otherwise 409 |
| Remove shelf | see below |
| Move a book to another shelf | target needs free space; also works for lent books, whose reserved slot moves along |

Open question: **what happens to the books when a shelf is removed?**

1. *Only empty shelves can be removed* (409 otherwise). Simplest and explicit; staff moves the books first.
2. *Remove and redistribute*: books move to other shelves with free space, all or nothing (409 if the remaining space is not enough). Lent books move too, so their return keeps working.

Proposal: start with option 1 plus the "move book" operation, so that staff decides where books go; option 2 can follow as a convenience on top.

### Multiple copies per ISBN

Today one ISBN is exactly one physical book.

- Split the *title* (ISBN, title, author, year) from the *copy* (own copy ID, shelf, reference-only, loan state). Reference-only becomes a property of the copy, e.g. one reference copy and two lendable ones.
- Loans reference a copy instead of an ISBN. Lending by ISBN picks any available lendable copy; staff can also lend or return a specific copy.
- Search returns titles with the number of available copies instead of a single lent flag.
- The data file gets format version 2; version 1 files are migrated to one copy per title on load.

### Authentication with accounts and sessions

Today static bearer tokens from `tokens.json` are the only credentials: no accounts, no passwords, no expiry, and anyone holding a token is that user until the file is changed. The token comparison is already hashed and constant-time; the rest should become:

- **Accounts:** members and employees stored with name, role and password hash (Argon2id via `golang.org/x/crypto`, which would be the first external dependency, or PBKDF2 from the standard library's `crypto/pbkdf2`). Never store or log plain passwords.
- **Login and sessions:** `POST /login` checks the password and issues a random session ID (32 bytes from `crypto/rand`); the server stores only its hash with user, expiry and last activity. `POST /logout` deletes it. Sessions expire after an absolute lifetime and after inactivity.
- **Cookies:** the browser gets the session as a cookie with `HttpOnly`, `Secure` and `SameSite=Strict`, so scripts cannot read it. Because cookies are sent automatically, write requests additionally need CSRF protection (checking the `Origin` header or a CSRF token). API clients can keep using bearer tokens.
- **Brute-force protection:** rate limiting and a temporary lockout after repeated failed logins, with the same error message for unknown users and wrong passwords.
- **Transport:** HTTPS only (TLS directly in the server or via a reverse proxy) plus HSTS; without it, passwords and sessions travel in clear text.
- **Later:** single sign-on (OIDC) for employees, password reset, and an audit log of who lent, returned or changed what.

## Structure

| Package | Responsibility |
|---|---|
| `internal/library` | domain: books, shelves, loans, search, snapshots |
| `internal/storage` | JSON file persistence |
| `internal/auth` | static bearer tokens, roles |
| `internal/api` | HTTP handlers, DTOs, access control, error mapping |
| `internal/web` | embedded single-page frontend |
| `cmd/server` | wiring, flags, sample data |
