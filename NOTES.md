# Notes

Things I learned, decisions I made, and mistakes I hit while building this.

## Mistakes

<!-- What went wrong, why, how I found it, what I'd do differently. -->

- used `FormValue` instead of `PathValue` when trying to retrieve id from "/events/{id}/reserve"
- missed leading slash in paths, causing the first segment to be treated as a domain name
- forgot to include `FOR UPDATE` in `SELECT` query in transaction

## Lessons & Decisions

### Go-isms

- use -er convention when naming single-method interfaces (eg. `Reserver`)
- compile-time interface checks via `var _ <interface> = (*<implementation>)(nil)`.
- compare errors with `errors.Is(error, error)`. important when errors are wrapped with `%w`

### Docker

- `docker compose up` & `docker compose down` for containers defined in `compose.yaml`
  - `docker compose up --detach`
  - `docker compose down --volumes`
- `docker compose ps` (ps = process status)
- `docker compose exec <container> <command>`

## Postgres

- single quotes for string literals, double quotes for column or table identifiers
- `\dt` (tables), `\di` (indexes), `\d <table>`
- `BEGIN` & `COMMIT` for transactions, used for atomicity

## Concurrency

<!-- M1 results go here: prediction vs. what actually happened, what -race reported -->
