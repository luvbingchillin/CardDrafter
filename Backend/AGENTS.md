# Go Backend Engineering Guidelines

## 1. Architecture & Layering
- **Routing & Server Hardening:**
  - Use Go 1.22+ standard library `http.NewServeMux` method+path patterns (e.g. `POST /api/packs/{setCode}/open`) and extract path parameters with `r.PathValue("setCode")`.
  - Always run HTTP traffic through a configured `&http.Server` with explicit `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`, paired with `signal.NotifyContext` (`SIGINT`/`SIGTERM`) and `srv.Shutdown(ctx)` so `defer` cleanup blocks execute cleanly.
  - Cap all JSON POST bodies with `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)` (1 MB) before decoding.
- **Consumer-Driven Interfaces:**
  - Follow *"Accept interfaces, return structs."* Declare narrow interfaces (`PackOpener`, `CardProvider`, `EventStore`) in the consuming package and enforce compile-time satisfaction via `var _ Interface = (*ConcreteType)(nil)`.
  - Guard against the **Typed-Nil Interface Trap**: never assign a `nil` concrete pointer to an interface variable, and guard receiver methods when underlying clients (`rdb`) are optional.
- **Layer Responsibilities:**
  - `internal/handlers/`: Transport parsing, JSON streaming (`json.NewEncoder(w).Encode`), and uniform JSON error responses (`writeJSONError`). Never leak raw internal `err.Error()` strings on `500 Internal Server Error` responses; log details via `log/slog` (`slog.Error`).
  - `internal/services/`: Transport-agnostic domain logic, double-checked locking caches (`PackService`), and worker-pool simulations (`SimulationService`).
  - `internal/repository/`: MongoDB (`go.mongodb.org/mongo-driver/v2`) and Redis (`github.com/redis/go-redis/v9`) persistence adapters.
  - `internal/draft/`: Encapsulated real-time multiplayer draft subsystem (`Hub`, `Room`, `Client`, `AggregateRoomState`, `DraftEvent`).

## 2. Event-Sourced Draft Engine Invariants
- **Single-Source-of-Truth Reducer:** `Room` embeds `agg *AggregateRoomState`. Command handlers must validate against `r.agg` and mutate state **exclusively** by calling `r.emit(eventType, payload)`, which appends the event to Redis Streams (`XADD` + 24h `EXPIRE`) and then calls `r.agg.Apply(event)`.
- **Strict Ledger Consistency:** If `r.eventStore.Append` fails, `r.emit` must abort before calling `r.agg.Apply(event)` so in-memory RAM never diverges from the persisted Redis stream.
- **Virtual Actor Passivation & Closed-Channel Broadcasts:**
  - Empty rooms passivate after 5 minutes of zero connected clients (`idleTimer`), removing themselves from `Hub` (`r.hub.RemoveRoom(r.id)`) and closing `r.done` (`defer close(r.done)`).
  - External goroutines sending to unbuffered room channels (`room.register`, `room.unregister`, `room.actionChan`) must always `select` against `<-room.Done()` to prevent deadlocks.
- **Presence Projections:** Keep `Seat.Client` tagged with `` `json:"-"` `` and maintain `Seat.IsConnected bool` for frontend JSON broadcasts.
