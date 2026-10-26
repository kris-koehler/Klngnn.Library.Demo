# Bücherei

## Deutsch

Umgesetzt ist ein Go-Backend ohne externe Abhängigkeiten, das eine Bücherei mit Regalen begrenzter Kapazität verwaltet und Suchen, Ausleihen, Zurückgeben und Anlegen von Büchern sowie die Regalbelegung über eine JSON-HTTP-API sowie ein schmales Browser-Frontend anbietet; Start mit `go run ./cmd/server`, danach http://localhost:8080 öffnen, Tests mit `go test ./...` (bzw. `go test -race ./...`, sofern cgo und ein C-Compiler verfügbar sind – auf der Entwicklungsmaschine war das nicht der Fall, der Nebenläufigkeitstest lief stattdessen mit `-count=200`).
Die Domäne (`internal/library`) kennt weder HTTP noch Dateien, den Ausleihstatus verwaltet zentral die Bücherei statt das Buch, und ausgeliehene Bücher behalten ihren Regalplatz, damit eine Rückgabe nie an einem vollen Regal scheitert.
Präsenzbestand ist eine Eigenschaft des Buchs statt einer Unterklasse, Fehler sind Sentinel-Errors, die die API auf 400/404/409 abbildet, und ein Mutex macht das Ausleihen bei parallelen Requests atomar.
Der Zustand wird nach jeder Änderung noch unter dem Lock atomar in `buecherei.json` geschrieben (Temp-Datei + Rename); schlägt das Speichern fehl, wird die Änderung zurückgerollt, und fehlt die Datei beim Start, werden Beispieldaten angelegt.
Die ISBN wird nur auf ihr Format geprüft, weil zwei der drei Beispiel-ISBNs eine ungültige Prüfziffer haben; „Clean Code“ ist zur Demonstration als Präsenzbestand markiert.
Als Nächstes: Prüfziffer-Validierung, mehrere Exemplare pro ISBN, Personenverwaltung mit Leihfristen, Authentifizierung für schreibende Endpunkte, eine Datenbank statt der JSON-Datei und weitere Endpunkte (Einzelabruf, Regale anlegen).

## English

A dependency-free Go backend that manages a library with capacity-limited shelves and offers searching, lending, returning and adding books as well as shelf occupancy via a JSON HTTP API and a minimal browser frontend; run it with `go run ./cmd/server`, open http://localhost:8080 and test it with `go test ./...` (or `go test -race ./...` where cgo and a C compiler are available – they were not on the development machine, so the concurrency test was run with `-count=200` instead).
The domain (`internal/library`) knows nothing about HTTP or files, loan state is kept centrally in the library rather than in the book, and lent books keep their shelf slot so a return can never fail because a shelf is full.
Reference-only is a property of a book instead of a subclass, errors are sentinel errors that the API maps to 400/404/409, and a mutex keeps lending atomic under concurrent requests.
After every change the state is written atomically to `buecherei.json` while the lock is still held (temp file + rename); if saving fails the change is rolled back, and if the file is missing on start, sample data is created.
ISBNs are only validated for format because two of the three sample ISBNs have an invalid check digit; "Clean Code" is marked as reference-only for demonstration.
Next steps: check-digit validation, multiple copies per ISBN, member management with due dates, authentication for write endpoints, a database instead of the JSON file and further endpoints (single lookup, add shelves).

## Server

```bash
go run ./cmd/server                                # :8080, data in ./buecherei.json
go run ./cmd/server -addr :9000 -data /tmp/b.json  # other port and data file
```

Delete the data file to start over with the sample data.

## API

| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/books?q=…` | – | 200, list of books |
| POST | `/books` | `{"title":"…","author":"…","isbn":"…","year":2000,"referenceOnly":false}` | 201, book incl. shelf |
| POST | `/books/{isbn}/loan` | `{"borrower":"…"}` | 201, loan |
| DELETE | `/books/{isbn}/loan` | – | 204 |
| GET | `/shelves` | – | 200, `[{"id":"A","capacity":2,"used":2,"free":0}]` (lent books count as used) |

Errors are returned as `{"error":"…"}` with status 400 (invalid input), 404 (unknown book), 409 (conflict, e.g. already lent, reference-only, duplicate ISBN or no free shelf space) or 500 (e.g. data file not writable).

```bash
curl -s 'localhost:8080/books?q=hobbit'
curl -si -X POST -H 'Content-Type: application/json' -d '{"borrower":"Max"}' \
  localhost:8080/books/978-3-608-93800-4/loan
curl -si -X DELETE localhost:8080/books/978-3-608-93800-4/loan
curl -si -X POST -H 'Content-Type: application/json' \
  -d '{"title":"Momo","author":"Michael Ende","isbn":"978-3-522-20210-1","year":1973}' \
  localhost:8080/books
```

## Structure

| Package | Responsibility |
|---|---|
| `internal/library` | domain: books, shelves, loans, search, snapshots |
| `internal/storage` | JSON file persistence |
| `internal/api` | HTTP handlers, DTOs, error mapping |
| `internal/web` | embedded single-page frontend |
| `cmd/server` | wiring, flags, sample data |
