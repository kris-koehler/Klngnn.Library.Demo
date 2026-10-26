# Bücherei

## Deutsch

Umgesetzt ist ein Go-Backend ohne externe Abhängigkeiten, das eine Bücherei mit Regalen begrenzter Kapazität verwaltet und Suchen, Ausleihen und Zurückgeben über eine JSON-HTTP-API anbietet; Start mit `go run ./cmd/server` (Port 8080), Tests mit `go test ./...` (bzw. `go test -race ./...`, sofern cgo und ein C-Compiler verfügbar sind – auf der Entwicklungsmaschine war das nicht der Fall, der Nebenläufigkeitstest lief stattdessen mit `-count=200`).
Die Domäne (`internal/library`) kennt kein HTTP, den Ausleihstatus verwaltet zentral die Bücherei statt das Buch, und ausgeliehene Bücher behalten ihren Regalplatz, damit eine Rückgabe nie an einem vollen Regal scheitert.
Präsenzbestand ist eine Eigenschaft des Buchs statt einer Unterklasse, Fehler sind Sentinel-Errors, die die API auf 400/404/409 abbildet, und ein Mutex macht das Ausleihen bei parallelen Requests atomar.
Die ISBN wird nur auf ihr Format geprüft, weil zwei der drei Beispiel-ISBNs eine ungültige Prüfziffer haben; „Clean Code“ ist zur Demonstration als Präsenzbestand markiert.
Als Nächstes: Prüfziffer-Validierung, mehrere Exemplare pro ISBN, Personenverwaltung mit Leihfristen, Persistenz und weitere Endpunkte (Buch anlegen, Regale anzeigen).

## English

A dependency-free Go backend that manages a library with capacity-limited shelves and offers search, lending and returning via a JSON HTTP API; run it with `go run ./cmd/server` (port 8080) and test it with `go test ./...` (or `go test -race ./...` where cgo and a C compiler are available – they were not on the development machine, so the concurrency test was run with `-count=200` instead).
The domain (`internal/library`) knows nothing about HTTP, loan state is kept centrally in the library rather than in the book, and lent books keep their shelf slot so a return can never fail because a shelf is full.
Reference-only is a property of a book instead of a subclass, errors are sentinel errors that the API maps to 400/404/409, and a mutex keeps lending atomic under concurrent requests.
ISBNs are only validated for format because two of the three sample ISBNs have an invalid check digit; "Clean Code" is marked as reference-only for demonstration.
Next steps: check-digit validation, multiple copies per ISBN, member management with due dates, persistence and further endpoints (add books, list shelves).

## API

| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/books?q=…` | – | 200, list of books |
| POST | `/books/{isbn}/loan` | `{"borrower":"…"}` | 201, loan |
| DELETE | `/books/{isbn}/loan` | – | 204 |

Errors are returned as `{"error":"…"}` with status 400 (invalid input), 404 (unknown book) or 409 (conflict, e.g. already lent or reference-only).

```bash
curl -s 'localhost:8080/books?q=hobbit'
curl -si -X POST -H 'Content-Type: application/json' -d '{"borrower":"Max"}' \
  localhost:8080/books/978-3-608-93800-4/loan
curl -si -X DELETE localhost:8080/books/978-3-608-93800-4/loan
```
