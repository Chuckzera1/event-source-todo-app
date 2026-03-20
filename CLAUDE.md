# CLAUDE.md — Event Source Todo API

This file guides AI agents working on this codebase. Read it fully before writing any code.

---

## 1. Project Overview

A learning project to study event sourcing in Go. The goal is to understand how domain events
drive state changes, how pub/sub (NATS) replaces direct service calls, and how Clean/Hexagonal
Architecture keeps infrastructure details from leaking into business logic.

**Learning goals:**
- Model state as a sequence of immutable domain events
- Use NATS as the event bus: commands publish events, projections consume them
- Keep domain logic free of framework, database, and network concerns
- Apply design patterns from refactoring.guru deliberately, not cargo-culted

**Readability is the primary quality bar.** If a performance optimization makes code harder to
read, do not apply it unless there is a measured, documented need. Prefer the obvious solution.

---

## 2. Architecture

### Directory Layout

```
event-source-todo-app/
├── cmd/
│   └── todo-api/
│       └── main.go       # Composition root. Wires everything together. No logic here.
│
└── internal/             # Not importable by external modules (Go compiler enforced).
    ├── domain/           # Core business logic. No imports from any other layer.
    │   ├── todo.go       # Aggregate root (Todo struct + methods)
    │   ├── events.go     # Domain event types and factory functions
    │   └── errors.go     # Sentinel errors (ErrTodoNotFound, etc.)
    │
    ├── application/      # Use cases, interfaces, DTOs. Imports internal/domain only.
    │   ├── ports/        # Interfaces: TodoRepository, EventPublisher, EventSubscriber
    │   ├── usecases/     # One file per use case: create_todo.go, complete_todo.go, etc.
    │   └── dto/          # Request/response structs for the API boundary
    │
    ├── infrastructure/   # Concrete adapters. Imports internal/application/ports and internal/domain.
    │   ├── nats/         # NATS publisher and subscriber implementations
    │   └── gorm/         # GORM repository implementation and DB models
    │
    └── api/              # HTTP handlers and NATS message routing. Thin layer only.
        ├── http/         # net/http or chi handlers, middleware chain
        └── messaging/    # NATS subscription handlers that invoke use cases
```

**`cmd/todo-api/`** holds the main package for the HTTP API binary. Add another subdirectory under
`cmd/` (e.g. `cmd/migrator/`) for each additional binary.

**`internal/`** enforces Go's `internal` rule: code in **other modules** cannot import these
packages. Any package **inside this repository** may import `internal/...` as long as it follows
the dependency rules below.

### Dependency Rule (strict — violations block merging)

```
domain  <--  application  <--  infrastructure
  ^               ^                   ^
  └───────────── api                  │
                  ^                   │
                  └────── cmd/todo-api/main.go ────┘
                  (main imports all internal layers to wire them together)
```

- `internal/domain` imports nothing from this project.
- `internal/application` imports `internal/domain` only.
- `internal/infrastructure` imports `internal/application/ports` and `internal/domain` (needed to
  implement port interfaces that use domain types). It must not contain any business logic — that
  belongs in `internal/domain`.
- `internal/api` imports `internal/application/ports` and `internal/application/dto` only.
- `cmd/todo-api/main.go` is the **only** place that instantiates infrastructure structs and wires
  them into interfaces.

The PR template layer `repository` is **not** a top-level directory. It maps to:
- the interfaces in `internal/application/ports/` (the abstraction)
- the GORM structs in `internal/infrastructure/gorm/` (the implementation)

---

## 3. Event Sourcing Flow

Every state change follows this exact sequence:

```
HTTP / NATS Request
        |
        v
  Use Case (Command Handler)          <- validates input, builds command
        |
        v
  Domain Aggregate (Todo)             <- applies business rules, emits []domain.Event
        |
        v
  Event Store (Repository)            <- appends events (never updates/deletes)
        |
        v
  Event Publisher (NATS)              <- publishes each event to a subject
        |
        v
  Event Subscribers (Projectors)      <- consume events, update read models
        |
        v
  Read Model / Query Store            <- used by query use cases
```

**Core invariants:**
- Events are immutable facts. Never update or delete event rows.
- Aggregates rebuild their state by replaying their event history.
- Read models are derived state — they can be rebuilt from the event log at any time.
- Commands may fail (return error). Events may not — if an event was stored, it happened.
- Event subjects use dot notation: `"todo.created"`, `"todo.completed"`.

---

## 4. Design Patterns

When you write code that matches a named pattern, add a comment on the relevant struct or
function: `// Pattern: Observer`, `// Pattern: Decorator`. This makes the learning intent
searchable in the codebase.

### Observer — Event Subscribers

Use for: NATS subscribers that react to domain events. Each subscriber has one responsibility.

```go
// internal/application/ports/event_subscriber.go

// Pattern: Observer
type EventHandler func(event domain.Event) error

type EventSubscriber interface {
    Subscribe(subject string, handler EventHandler) error
}
```

Each concrete subscriber (`TodoProjector`, `AuditLogSubscriber`) is a separate struct registered
at startup in `cmd/todo-api/main.go`. They do not know about each other.

### Decorator — Cross-Cutting Concerns

Use for: wrapping use cases or repositories with logging, validation, or metrics without
modifying the core implementation.

```go
// internal/infrastructure/gorm/logging_repository.go

// Pattern: Decorator
type LoggingRepository struct {
    logger log.Logger
    next   ports.TodoRepository   // accepts the interface, not the concrete type
}

func (r LoggingRepository) Save(ctx context.Context, todo domain.Todo) error {
    r.logger.Info("saving todo", "id", todo.ID)
    return r.next.Save(ctx, todo)
}
```

Stack decorators in `cmd/todo-api/main.go` only:

```go
var repo ports.TodoRepository
repo = gorm.NewTodoRepository(db)
repo = gorm.NewLoggingRepository(logger, repo)   // Pattern: Decorator
```

### Repository — Persistence Abstraction

Interface lives in `internal/application/ports`. It uses domain types, never GORM models. The GORM
model is an internal detail of `internal/infrastructure/gorm` and is never exported outside that
package.

```go
// internal/application/ports/todo_repository.go
type TodoRepository interface {
    Save(ctx context.Context, todo domain.Todo) error
    FindByID(ctx context.Context, id uuid.UUID) (domain.Todo, error)
    FindAll(ctx context.Context) ([]domain.Todo, error)
}
```

### Factory — Creating Domain Events and Aggregates

Use for: constructing domain events with all required fields in one place.

```go
// internal/domain/events.go

// Pattern: Factory
func NewTodoCreated(todoID uuid.UUID, title string, occurredAt time.Time) TodoCreated {
    return TodoCreated{
        EventID:     uuid.New(),
        OccurredAt:  occurredAt,
        AggregateID: todoID,
        Title:       title,
    }
}
```

Never construct event structs with inline literals outside the `internal/domain` package.

Always pass `occurredAt` in from the caller (typically the use case, which receives it from an
injected clock). This keeps domain logic deterministic and easy to test.

### Command — Use Cases as Command Handlers

One use case file = one command struct + one handler struct + one `Execute` method.

```go
// internal/application/usecases/create_todo.go

// Pattern: Command
type CreateTodoCommand struct {
    Title string
}

type CreateTodoResult struct {
    TodoID uuid.UUID
}

type CreateTodoHandler struct {
    repo      ports.TodoRepository
    publisher ports.EventPublisher
}

func NewCreateTodoHandler(repo ports.TodoRepository, pub ports.EventPublisher) CreateTodoHandler {
    return CreateTodoHandler{repo: repo, publisher: pub}
}

func (h CreateTodoHandler) Execute(ctx context.Context, cmd CreateTodoCommand) (CreateTodoResult, error) {
    // ...
}
```

### Strategy — Interchangeable Behaviors

Use for: pluggable behaviors like event serialization (JSON today, Protobuf later).

```go
// internal/application/ports/event_serializer.go

// Pattern: Strategy
type EventSerializer interface {
    Marshal(event domain.Event) ([]byte, error)
    Unmarshal(data []byte, eventType string) (domain.Event, error)
}
```

The NATS publisher accepts `EventSerializer` as a constructor parameter. Never hardcode `json.Marshal`.

---

## 5. Go Conventions

### Interfaces

- Define interfaces in the **consumer's** package (`internal/application/ports`), not the implementor's.
- Keep interfaces small. Prefer one-method interfaces where possible.
- Name single-method interfaces after the method: `Publisher`, `Subscriber`, `Finder`.
- Do not embed interfaces in structs to partially satisfy them. Implement fully.
- Compose small interfaces into larger ones only when genuinely needed.

### Naming

- Exported types: `PascalCase`. Unexported: `camelCase`.
- Acronyms in exported names: `ID` not `Id`, `HTTP` not `Http`, `NATS` not `Nats`.
- Constructors: `NewXxx(deps...) Xxx` — return the concrete type, not the interface.
- Test files: `xxx_test.go`. Use `package xxx_test` for black-box tests.
- Avoid stuttering: `domain.Event` not `domain.DomainEvent`; `ports.Publisher` not `ports.PublisherInterface`.

### Error Handling

- Never discard errors. `_ = f()` is forbidden unless the function is documented as always-nil.
- Wrap errors with context: `fmt.Errorf("create todo: %w", err)`.
- Sentinel errors in `internal/domain/errors.go`: `var ErrTodoNotFound = errors.New("todo not found")`.
- Check errors with `errors.Is` / `errors.As`, never by string comparison.
- Return `(T, error)`. Never return a zero-value `T` as a success sentinel.

### Structs and Functions

- Use functional options for optional configuration:
  ```go
  type ServerOption func(*Server)
  func WithTimeout(d time.Duration) ServerOption { ... }
  ```
- No naked returns. Always name what you are returning in the `return` statement.
- No `init()` functions. Initialize explicitly in `cmd/todo-api/main.go`.
- Embed only for true "is-a" relationships. Prefer composition with named fields.

### Contexts

- Every function that does I/O (DB, NATS, HTTP) accepts `context.Context` as its first parameter.
- Never store a context in a struct field.
- Pass context down. Never create a new `context.Background()` in the middle of a call stack.

---

## 6. Code Style Rules

1. **Readable over clever.** If you need a comment to explain what code does, first try to
   rename the variable or extract a function with a descriptive name.
2. **Explicit over implicit.** No magic, no reflection tricks, no `interface{}`/`any` unless
   the type system genuinely cannot express it.
3. **No premature optimization.** Do not add caching, pooling, or concurrency before there is
   a benchmark proving it is needed.
4. **One responsibility per file.** A file that does two unrelated things should be two files.
5. **Flat is better than deeply nested.** Use early returns to reduce nesting. Three levels of
   indent is a warning sign — extract a function.
6. **No global state.** No `var` at package level except sentinel errors and read-only constants.
   All dependencies are injected via constructors.
7. **Comments explain why, not what.** Do not write `// increment counter` above `count++`.
   Do write why a workaround exists or why a design decision was made.

---

## 7. Testing

### Commands

```bash
# Run all unit tests (default, no infrastructure required)
gotestsum --format short -- -count=1 ./...

# Verbose output
gotestsum --format standard-verbose -- -count=1 ./...

# Run integration tests (requires NATS + DB running)
gotestsum --format short -- -count=1 -tags=integration ./...

# Single package
gotestsum --format short -- -count=1 ./internal/domain/...
```

### Strategy Per Layer

| Layer | Type | Notes |
|---|---|---|
| `internal/domain` | Pure unit tests | No mocks, no I/O. Test aggregate behavior by applying events. |
| `internal/application` | Unit tests with fakes | Write simple hand-rolled fakes, not generated mocks. |
| `internal/infrastructure` | Integration tests | Tag with `//go:build integration`. Require real DB/NATS. |
| `internal/api` | Handler tests | Use `net/http/httptest`. Test routing and JSON shape only. |

### Test Naming

```go
func TestCreateTodoHandler_Execute_ShouldReturnID_WhenTitleIsValid(t *testing.T) {}
func TestTodo_Complete_ShouldReturnError_WhenAlreadyCompleted(t *testing.T) {}
```

Pattern: `Test[Type]_[Method]_Should[Outcome]_When[Condition]`

### Table-Driven Tests

Use table-driven tests whenever testing the same behavior with multiple inputs:

```go
tests := []struct {
    name    string
    cmd     CreateTodoCommand
    wantErr bool
}{
    {name: "valid title", cmd: CreateTodoCommand{Title: "Buy milk"}, wantErr: false},
    {name: "empty title", cmd: CreateTodoCommand{Title: ""}, wantErr: true},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // ...
    })
}
```

### What to Mock

Mock at the `internal/application/ports` interface boundary only. Do not mock domain types.
Write hand-rolled fakes for small interfaces (under 5 methods). They are easier to read
and maintain than generated mocks.

---

## 8. Anti-Patterns to Avoid

| Anti-pattern | Why it is wrong | What to do instead |
|---|---|---|
| Anemic domain model | Business logic ends up in use cases; domain is just data bags | Move validation and state transitions into aggregate methods |
| God use case | One handler does create, update, delete, and list | One file, one command handler per operation |
| Wrong layer imports | `internal/application` importing `internal/infrastructure` breaks dependency inversion | Define an interface in `internal/application/ports`; inject the concrete in `cmd/todo-api/main.go` |
| `interface{}` / `any` for events | Loses type safety at the core of the system | Use a `domain.Event` interface with `EventType() string` and `AggregateID() uuid.UUID` |
| Mutating events after creation | Events are immutable facts | Make event fields unexported or document the struct as read-only |
| `time.Now()` inside domain logic | Makes tests time-dependent and non-deterministic | Pass `occurredAt time.Time` into factory functions; inject a clock |
| Skipping context propagation | Breaks cancellation and tracing | Always accept and pass `ctx context.Context` as the first argument in I/O functions |
| Large interfaces in ports | Hard to fake in tests; violates interface segregation | Split into focused interfaces; compose them only when needed |
| `init()` functions | Hidden initialization order; hard to test | Explicit wiring in `cmd/todo-api/main.go` only |
| Panic in library code | Callers cannot recover gracefully | Return errors. Reserve `panic` for nil required dependencies caught at startup. |

---

## 9. Commands

```bash
# Build the binary
go build -o bin/todo-api ./cmd/todo-api/

# Run static analysis
go vet ./...

# Tidy dependencies
go mod tidy

# Lint (if golangci-lint is installed)
golangci-lint run ./...
```

---

## 10. PR Guidelines

All pull requests use `.github/pull_request_template.md`. Check all that apply in the template.

**Before opening a PR:**
- [ ] All unit tests pass: `gotestsum --format short -- -count=1 ./...`
- [ ] `go vet ./...` produces no output
- [ ] No layer imports a layer it should not (review Section 2 dependency rules)
- [ ] New use cases have a matching `Command` struct and `Handler` struct
- [ ] New domain events have a factory function in `internal/domain/events.go`
- [ ] New interfaces are defined in `internal/application/ports`, not in `internal/infrastructure` or `internal/api`
- [ ] Design patterns used are marked with a `// Pattern: Xxx` comment
- [ ] No global variables were added (other than sentinel errors)

**PR size:** Prefer small, vertical slices. A PR that adds one use case end-to-end
(`internal/domain` event → use case → repository method → `internal/api` HTTP handler) is the right size.
