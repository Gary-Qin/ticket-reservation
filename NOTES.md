# Notes

Things I learned, decisions I made, and mistakes I hit while building this.

## Mistakes

<!-- What went wrong, why, how I found it, what I'd do differently. -->

- initially used `FormValue` instead of `PathValue` when trying to retrieve id from "/events/{id}/reserve"
- initially missed leading slash in paths, causing the first segment to be treated as a domain name

## Lessons & Decisions

### Go-isms

- use -er convention when naming single-method interfaces (eg. `Reserver`)
- compile-time interface checks via `var _ <interface> = (*<implementation>)(nil)`.
- compare errors with `errors.Is(error, error)`. important when errors are wrapped with `%w`

## Concurrency

<!-- M1 results go here: prediction vs. what actually happened, what -race reported -->
