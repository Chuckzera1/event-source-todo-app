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

**Module path:** `github.com/Chuckzera1/event-source-todo-app` (see `go.mod`). **Go version:** `1.26.1`.

---

## 2. Architecture

### Directory Layout (current codebase)

```
event-source-todo-app/
├── cmd/
│   └── todo-api/
│       └── main.go       # Entry point: starts HTTP server (Gin). Keep free of business logic.
│
└── internal/
    ├── domain/                    # Core model. No imports from other internal packages.
    │   ├── task.go                # Task aggregate / entity
    │   ├── event.go               # Event shape (evolving toward full event sourcing)
    │   └── errors.go              # Sentinel errors (add when needed, e.g. ErrTaskNotFound)
    │
    ├── application/               # Use cases, repository interfaces, DTOs. Imports domain only.
    │   ├── repositories/          # Persistence ports (e.g. ITaskRepository)
    │   ├── usecases/              # One file per use case when added
    │   └── dto/                   # Request/response for API boundary
    │
    ├── di/                        # Composition: wires concrete infrastructure to application interfaces
    │   └── task.go                # Example: TaskRepository DI
    │
    ├── infrastructure/            # Adapters: DB, NATS, etc.
    │   ├── gorm.go                # package infrastructure — NewGorm(DSN) opens PostgreSQL via GORM
    │   ├── nats/                  # NATS adapters (when added)
    │   └── gorm/
    │       └── gormrepo/          # GORM repository implementations per aggregate
    │           └── task/          # e.g. model.go (TaskModel), create.go (CreateTaskRepositoryImpl)
    │
    └── api/                       # Target: thin HTTP / messaging layer (not wired yet)
        ├── http/                  # Prefer handlers here over growing main.go
        └── messaging/             # NATS handlers that call use cases
```

**`cmd/todo-api/`** holds the main package for the HTTP API binary. Add another subdirectory under
`cmd/` (e.g. `cmd/migrator/`) for each additional binary.

**`internal/`** enforces Go's `internal` rule: code in **other modules** cannot import these
packages. Any package **inside this repository** may import `internal/...` as long as it follows
the dependency rules below.

**Current bootstrap note:** `cmd/todo-api/main.go` currently only registers Gin and `/health`. The
event store, NATS publisher/subscribers, and DI constructors are **not** fully wired in `main` yet.
Treat Sections 3–4 as the **target** event-sourcing flow unless code exists for each step.

**Naming note:** Repository interfaces today live under `internal/application/repositories`. Some
guides call this folder `ports/`; either name is fine if the team picks one consistently.

### Dependency Rule (strict — violations block merging)

```
domain  <--  application  <--  infrastructure
  ^               ^                   ^
  └───────────── api (when present)   │
                  ^                   │
                  ├── cmd/todo-api/main.go
                  └── internal/di (composition — may import application + infrastructure)
```

- `internal/domain` imports nothing from this project.
- `internal/application` imports `internal/domain` only.
- `internal/infrastructure` imports `internal/application/repositories` and `internal/domain` (to
  implement repository interfaces with domain types). It must not contain business logic — that
  belongs in `internal/domain`.
- `internal/api` (when added) imports `internal/application/repositories`, `internal/application/dto`,
  and related application packages only — not `internal/infrastructure`.
- **Composition root:** `cmd/todo-api/main.go` **and** `internal/di`. Only these (or future
  bootstrap packages under `cmd/` / `internal/di`) may construct concrete infrastructure types and
  assign them to application interfaces. Prefer keeping `main` thin by delegating wiring to
  `internal/di` as it grows.

The PR template checkbox **repository** is **not** a top-level directory. It maps to:

- Interfaces in `internal/application/repositories/` (abstraction; use cases depend on these)
- GORM models and repository structs under `internal/infrastructure/gorm/gormrepo/...` (implementation;
  maps domain entities to persistence; never leak GORM models outside infrastructure)

---

## 3. Event Sourcing Flow

**Target flow** (implement end-to-end as the project grows; not all steps exist in `main` yet — see
Section 2 bootstrap note):

Every state change follows this exact sequence:

```
HTTP / NATS Request
        |
        v
  Use Case (Command Handler)          <- validates input, builds command
        |
        v
  Domain Aggregate (e.g. Task)        <- applies business rules, emits []domain.Event
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
- Event subjects use dot notation: e.g. `"task.created"`, `"task.completed"` (align names with your
  aggregate and ubiquitous language).

---

## 4. Design Patterns

When you write code that matches a named pattern, add a comment on the relevant struct or
function: `// Pattern: Observer`, `// Pattern: Decorator`. This makes the learning intent
searchable in the codebase.

### Observer — Event Subscribers

Use for: NATS subscribers that react to domain events. Each subscriber has one responsibility.

```go
// internal/application/repositories/event_subscriber.go (add when NATS subscribers exist; or a small ports file if split later)

// Pattern: Observer
type EventHandler func(event domain.Event) error

type EventSubscriber interface {
    Subscribe(subject string, handler EventHandler) error
}
```

Each concrete subscriber (`TaskProjector`, `AuditLogSubscriber`, etc.) is a separate struct registered
at startup in `cmd/todo-api/main.go` and/or `internal/di`. They do not know about each other.

### Decorator — Cross-Cutting Concerns

Use for: wrapping use cases or repositories with logging, validation, or metrics without
modifying the core implementation.

```go
// internal/infrastructure/gorm/logging_repository.go (example)

// Pattern: Decorator
type LoggingTaskRepository struct {
    logger log.Logger
    next   irepositories.ITaskRepository   // accepts the interface, not the concrete type
}

func (r LoggingTaskRepository) CreateTask(ctx context.Context, task domain.Task) error {
    r.logger.Info("creating task", "title", task.Title)
    return r.next.CreateTask(ctx, task)
}
```

Stack decorators in the composition root only (`cmd/todo-api/main.go` and/or `internal/di`):

```go
var repo irepositories.ITaskRepository
base := &rtask.CreateTaskRepositoryImpl{DB: db}   // concrete from gorm/gormrepo/task
repo = gorm.NewLoggingTaskRepository(logger, base) // Pattern: Decorator
```

### Repository — Persistence Abstraction

Interface lives in `internal/application/repositories`. It uses domain types, never GORM models. The
GORM model is an internal detail under `internal/infrastructure/gorm/gormrepo/...` and is not part
of the application API.

```go
// internal/application/repositories/task.go
type ITaskRepository interface {
    CreateTask(ctx context.Context, task domain.Task) error
}
```

Add methods such as `FindByID`, `FindAll` on the same interface or smaller interfaces as needs grow.

### Factory — Creating Domain Events and Aggregates

Use for: constructing domain events with all required fields in one place.

```go
// internal/domain/events.go (or event_types.go when you split from event.go)

// Pattern: Factory
func NewTaskCreated(taskID uuid.UUID, title string, occurredAt time.Time) TaskCreated {
    return TaskCreated{
        EventID:     uuid.New(),
        OccurredAt:  occurredAt,
        AggregateID: taskID,
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
// internal/application/usecases/create_task.go

// Pattern: Command
type CreateTaskCommand struct {
    Title string
}

type CreateTaskResult struct {
    TaskID string
}

type CreateTaskHandler struct {
    repo      irepositories.ITaskRepository
    publisher EventPublisher // define alongside other messaging ports when NATS exists
}

func NewCreateTaskHandler(repo irepositories.ITaskRepository, pub EventPublisher) CreateTaskHandler {
    return CreateTaskHandler{repo: repo, publisher: pub}
}

func (h CreateTaskHandler) Execute(ctx context.Context, cmd CreateTaskCommand) (CreateTaskResult, error) {
    // ...
}
```

### Strategy — Interchangeable Behaviors

Use for: pluggable behaviors like event serialization (JSON today, Protobuf later).

```go
// internal/application/repositories/event_serializer.go (example path)

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

- Define interfaces in the **consumer's** package (`internal/application/repositories`, or a
  dedicated `ports` package if the team consolidates naming), not the implementor's.
- Keep interfaces small. Prefer one-method interfaces where possible.
- Name single-method interfaces after the method: `Publisher`, `Subscriber`, `Finder`.
- Do not embed interfaces in structs to partially satisfy them. Implement fully.
- Compose small interfaces into larger ones only when genuinely needed.

### Naming

- Exported types: `PascalCase`. Unexported: `camelCase`.
- Acronyms in exported names: `ID` not `Id`, `HTTP` not `Http`, `NATS` not `Nats`.
- Constructors: `NewXxx(deps...) Xxx` — return the concrete type, not the interface.
- Test files: `xxx_test.go`. Use `package xxx_test` for black-box tests.
- Avoid stuttering: `domain.Event` not `domain.DomainEvent`; prefer `TaskRepository` over
  `TaskRepositoryInterface`.
- **Current codebase:** repository interfaces use the `I` prefix (e.g. `ITaskRepository`). Idiomatic Go
  often omits `I` (`TaskRepository` for the interface type). Prefer dropping `I` for **new** interfaces;
  rename existing ones when you are already touching call sites.

### Error Handling

- Never discard errors. `_ = f()` is forbidden unless the function is documented as always-nil.
- Wrap errors with context: `fmt.Errorf("create task: %w", err)`.
- Sentinel errors in `internal/domain/errors.go` when introduced: e.g.
  `var ErrTaskNotFound = errors.New("task not found")`.
- Check errors with `errors.Is` / `errors.As`, never by string comparison.
- Return `(T, error)`. Never return a zero-value `T` as a success sentinel.

### Structs and Functions

- Use functional options for optional configuration:
  ```go
  type ServerOption func(*Server)
  func WithTimeout(d time.Duration) ServerOption { ... }
  ```
- No naked returns. Always name what you are returning in the `return` statement.
- No `init()` functions. Initialize explicitly in `cmd/todo-api/main.go` and/or `internal/di`.
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

| Layer                     | Type                  | Notes                                                         |
| ------------------------- | --------------------- | ------------------------------------------------------------- |
| `internal/domain`         | Pure unit tests       | No mocks, no I/O. Test aggregate behavior by applying events. |
| `internal/application`   | Unit tests with fakes | Write simple hand-rolled fakes, not generated mocks.           |
| `internal/infrastructure` | Integration tests     | Tag with `//go:build integration`. Require real DB/NATS.      |
| `internal/api`            | Handler tests         | Use `net/http/httptest`. Test routing and JSON shape only.    |

### Test Naming

```go
func TestCreateTaskHandler_Execute_ShouldReturnID_WhenTitleIsValid(t *testing.T) {}
func TestTask_Complete_ShouldReturnError_WhenAlreadyCompleted(t *testing.T) {}
```

Pattern: `Test[Type]_[Method]_Should[Outcome]_When[Condition]`

### Table-Driven Tests

Use table-driven tests whenever testing the same behavior with multiple inputs:

```go
tests := []struct {
    name    string
    cmd     CreateTaskCommand
    wantErr bool
}{
    {name: "valid title", cmd: CreateTaskCommand{Title: "Buy milk"}, wantErr: false},
    {name: "empty title", cmd: CreateTaskCommand{Title: ""}, wantErr: true},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // ...
    })
}
```

### What to Mock

Mock at the `internal/application/repositories` interface boundary only. Do not mock domain types.
Write hand-rolled fakes for small interfaces (under 5 methods). They are easier to read
and maintain than generated mocks.

---

## 8. Anti-Patterns to Avoid

| Anti-pattern                     | Why it is wrong                                                                        | What to do instead                                                                                 |
| -------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Anemic domain model              | Business logic ends up in use cases; domain is just data bags                          | Move validation and state transitions into aggregate methods                                       |
| God use case                     | One handler does create, update, delete, and list                                      | One file, one command handler per operation                                                        |
| Wrong layer imports              | `internal/application` importing `internal/infrastructure` breaks dependency inversion | Define an interface in `internal/application/repositories`; inject the concrete in `cmd/todo-api/main.go` and/or `internal/di` |
| `interface{}` / `any` for events | Loses type safety at the core of the system                                            | Use a `domain.Event` interface with `EventType() string` and `AggregateID() uuid.UUID`             |
| Mutating events after creation   | Events are immutable facts                                                             | Make event fields unexported or document the struct as read-only                                   |
| `time.Now()` inside domain logic | Makes tests time-dependent and non-deterministic                                       | Pass `occurredAt time.Time` into factory functions; inject a clock                                 |
| Skipping context propagation     | Breaks cancellation and tracing                                                        | Always accept and pass `ctx context.Context` as the first argument in I/O functions                |
| Large interfaces in repositories | Hard to fake in tests; violates interface segregation                                  | Split into focused interfaces; compose them only when needed                                       |
| `init()` functions               | Hidden initialization order; hard to test                                              | Explicit wiring in `cmd/todo-api/main.go` and/or `internal/di` only                                |
| Panic in library code            | Callers cannot recover gracefully                                                      | Return errors. Reserve `panic` for nil required dependencies caught at startup.                    |

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
- [ ] New domain events have a factory function in `internal/domain` (e.g. `events.go` or dedicated event type files)
- [ ] New repository interfaces live in `internal/application/repositories`, not in `internal/infrastructure` or `internal/api`
- [ ] Design patterns used are marked with a `// Pattern: Xxx` comment
- [ ] No global variables were added (other than sentinel errors)

**PR template:** `.github/pull_request_template.md` lists **repository** as an affected layer — that means
**interfaces in `internal/application/repositories` plus GORM implementations under
`internal/infrastructure/gorm/gormrepo/...`**, not a separate top-level folder.

**PR size:** Prefer small, vertical slices. A PR that adds one use case end-to-end
(`internal/domain` event → use case → repository method → HTTP handler in `internal/api/http` or
equivalent) is the right size.

---

## 11. Pattern backlog (suggested improvements)

The following items are **not** required changes; they are ranked by **concern level** so the team can
decide what to tackle next (**10 = highest concern**). See Section 2 for the authoritative layout.

| Level | Suggestion |
| ----- | ---------- |
| **10** | `domain.Event` with `Data any` conflicts with typed event sourcing. Prefer a discriminated `Event` interface plus factories in `internal/domain`, or typed variants per event. |
| **9** | Aggregate identity: `domain.Task.ID` vs repository generating `uuid.New()` on persist must be aligned so state and replay stay consistent. |
| **8** | `TaskModel` embedding `gorm.Model` while redefining `ID` as `uuid.UUID` risks conflicting GORM fields; prefer one approach (explicit columns + UUID PK, or documented composite pattern). |
| **7** | Single story for composition: `cmd/todo-api/main.go` vs `internal/di` — both are valid; document which owns which wiring so there is no ambiguous second “root”. |
| **6** | Unify naming between `internal/application/repositories` and a possible `ports/` folder to reduce cognitive load and PR checklist drift. |
| **5** | HTTP: Gin currently lives in `main`; as routes grow, move handlers to `internal/api/http` and keep `main` as a thin bootstrap. |
| **4** | Drop the `I` prefix on interfaces (`ITaskRepository` → `TaskRepository`) when refactoring touch points. |
| **4** | Prefer package name `repositories` without an `irepositories` import alias where possible. |
| **3** | `TaskRepositoryDI` embedding the interface is uncommon; consider functions that return `ITaskRepository` directly unless wrapping for decorators. |
| **2** | On Windows, avoid duplicate path styles in Git (`cmd/todo-api` vs `cmd\todo-api`) when adding files. |
| **1** | Align `.github/pull_request_template.md` wording with “repository = application interfaces + infrastructure GORM” to remove ambiguity. |
