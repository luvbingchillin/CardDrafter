# Go Backend Engineering Guidelines

## Architecture & Conventions
- **Routing:** Use Go 1.22+ standard library `http.NewServeMux` route patterns (e.g. `POST /api/packs/{setCode}/open`). Extract parameters using `r.PathValue(...)`.
- **Layering:** Maintain clean separation across:
  - `handlers/`: Request parsing, JSON streaming via `json.NewEncoder(w).Encode(...)`, HTTP status codes.
  - `services/`: Business logic, MTG pack collation, in-memory caching.
  - `repository/`: MongoDB queries using the official driver (`go.mongodb.org/mongo-driver/v2`).
  - `models/`: Plain Go structs with `bson` and `json` tags.

## Concurrency & Data Safety
- Always protect in-memory maps or caches accessed across Goroutines using `sync.RWMutex` (`RLock`/`RUnlock` for reads, `Lock`/`Unlock` for writes).
- Never ignore errors; check and handle every `if err != nil`.
- Pass `context.Context` through handlers down to database operations.
