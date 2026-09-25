# qpub-go architecture

This SDK follows the same **architectural intent** as [qpub-js](https://github.com/qpubio/qpub-js): clear boundaries between protocol, ports, application logic, infrastructure, and the public facade. It is **not** backend-style DDD (no domain/application/infrastructure trees like qpub-backend). Structure is **Go-idiomatic**: packages by concern, `internal/` for non-public wiring, interfaces for testability, one composition root.

## Goals

- **Isolate layers** so protocol and use cases do not depend on HTTP/WebSocket details.
- **Scale** by adding features inside the right package without reshaping the module.
- **Avoid circular imports** via dependency direction and small port interfaces.
- **Match qpub-js mentally** (Socket/Rest, managers, auth, channels, queues) without copying JS patterns (DI container, EventEmitter, Promises).

## Layer model

```mermaid
flowchart TB
  subgraph api [API layer]
    QPubPkg[qpub]
  end
  subgraph app [Application layer]
    Auth[auth]
    Conn[connection]
    Ch[channel]
    Que[queue]
  end
  subgraph kernel [Shared kernel]
    Proto[protocol]
    Opt[option]
    Ev[events]
  end
  subgraph infra [Infrastructure]
    HTTP[transport/httpclient]
    WS[transport/ws]
    InternalJWT[internal/jwt]
    InternalCrypto[internal/crypto]
    InternalAPIKey[internal/apikey]
  end
  subgraph support [Internal support]
    Log[internal/logger]
    Emit[internal/emitter]
  end
  QPubPkg --> Auth
  QPubPkg --> Conn
  QPubPkg --> Ch
  QPubPkg --> Que
  QPubPkg --> Opt
  Auth --> Opt
  Auth --> Proto
  Conn --> Ch
  Conn --> Auth
  Conn --> WS
  Ch --> Proto
  Ch --> WS
  Ch --> HTTP
  Que --> Proto
  Que --> HTTP
  Auth --> InternalJWT
  Auth --> InternalCrypto
  Auth --> InternalAPIKey
  Auth --> HTTP
  app --> Ev
  app --> support
  kernel --> kernel
```

### Layer definitions

| Layer                   | Packages                                                            | Responsibility                                                                                                                                                      |
| ----------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **API**                 | `qpub/`                                                             | Only stable entry points: `NewSocket`, `NewRest`, re-exported types, event name constants, consumer-facing interfaces. Wires dependencies (composition root).       |
| **Application**         | `auth/`, `connection/`, `channel/`, `queue/`                        | Use cases: authenticate, connect/reconnect, subscribe/publish, enqueue/worker loop. Depends on **ports** (HTTP, WS, options, logger) and **kernel**, not on `qpub`. |
| **Shared kernel**       | `protocol/`, `option/`, `events/`                                   | Wire DTOs, config defaults, event names/payload shapes. No I/O, no managers.                                                                                        |
| **Infrastructure**      | `transport/*`, `internal/jwt`, `internal/crypto`, `internal/apikey` | Real HTTP/WebSocket, crypto, JWT. Implements behavior behind small interfaces where possible.                                                                       |
| **Internal support**    | `internal/logger`, `internal/emitter`                               | Cross-cutting helpers; not imported by `protocol/` or `option/`.                                                                                                    |
| **Testing (consumers)** | `testing/`                                                          | Mocks and test constructors; may import `qpub` and `option`.                                                                                                        |
| **Examples**            | `examples/`                                                         | Sample programs; import `qpub` only.                                                                                                                                |

## Comparison with qpub-js (spirit, not shape)

| qpub-js                                              | qpub-go                                                                 |
| ---------------------------------------------------- | ----------------------------------------------------------------------- |
| `types/protocol`, `types/config`                     | `protocol/`, `option/`                                                  |
| `types/services` (ports)                             | `qpub/interfaces.go` today; target: `internal/port/` as interfaces grow |
| `core/managers`, `core/channels`, `core/connections` | `auth/`, `channel/`, `connection/`, `queue/`                            |
| `core/transport`, `core/shared`                      | `transport/*`, `internal/*`                                             |
| `core/bootstrap` (DI)                                | `qpub/socket.go`, `qpub/rest.go` (composition root)                     |
| `qpub.ts` facade                                     | `qpub/` package                                                         |

## Import rules (dependency direction)

Allowed imports flow **downward** only. If a change requires an upward import, introduce or extend a **port interface** in the kernel/application boundary instead.

| From                              | May import                                                                      | Must not import                                                                 |
| --------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `protocol`, `option`, `events`    | std library only                                                                | `auth`, `channel`, `connection`, `queue`, `transport`, `qpub`, `testing`        |
| `internal/*`                      | std, `option` (logger only if needed)                                           | `qpub`, `auth`, `channel`, `connection`, `queue`                                |
| `transport/*`                     | std, `internal/logger` (optional)                                               | `qpub`, `auth`, `channel`, `connection`, `queue`                                |
| `auth`, `queue`, `channel` (REST) | `option`, `protocol`, `events`, `internal/*`, `internal/port` (HTTP)            | `qpub`, `connection` (REST managers only; socket channel uses `MessageSender`)  |
| `connection`                      | `option`, `protocol`, `events`, `auth`, `channel`, `transport/ws`, `internal/*` | `qpub`, `queue`                                                                 |
| `channel`                         | `option`, `protocol`, `events`, `auth`, `transport/*`, `internal/*`             | `qpub`, `connection` (avoid: connection orchestrates channels, not the reverse) |
| `qpub`                            | all application + kernel + transport packages used for wiring                   | `testing`                                                                       |
| `testing`                         | `qpub`, `option`, `auth`, … for mocks                                           | —                                                                               |
| `examples/*`                      | `qpub`, `channel` (subscribe options), std                                      | `internal/*`, `transport/*` directly                                            |

**Circular import avoidance in Go**

1. **Connection ↔ channel**: `connection` imports `channel` for resubscribe/dispatch; `channel` must not import `connection`. Socket lifecycle stays in `connection`.
2. **Auth ↔ HTTP**: prefer `auth.HTTPDoer` (small interface) over concrete `httpclient.Client` in tests and future refactors.
3. **Shared types**: keep wire types in `protocol/`; do not duplicate DTOs in `auth` or `queue`.
4. **New subpackages**: if two application packages need each other, extract shared contracts to `protocol/`, `events/`, or `internal/port/`.

## Composition root

All runtime wiring for `Socket` and `Rest` lives in:

- [qpub/socket.go](../qpub/socket.go)
- [qpub/rest.go](../qpub/rest.go)

New dependencies (custom HTTP client, logger, clock) should be added here (or in a future `internal/bootstrap/` package called only from `qpub`), not scattered across application packages.

## Public surface

Consumers should depend on:

```go
import "github.com/qpubio/qpub-go/qpub"
```

Advanced testing:

```go
import "github.com/qpubio/qpub-go/testing"
```

Do not document or encourage importing `internal/*` or `transport/*` from application code outside this module.

## Concurrency conventions

- **Per-channel message handlers**: run serially for a given `SocketChannel` (same handler queue as qpub-js single-thread semantics) unless documented otherwise.
- **Connection read loop**: one goroutine demuxes WebSocket frames; dispatches to channel manager.
- **Lifecycle**: prefer `context.Context` for cancel; `Reset()` on Socket/Rest mirrors qpub-js teardown order: connection → channels → auth → options.

## Current layout vs target (evolution)

Today’s repo is **mostly aligned** with the layer names above but some **application packages still call infrastructure directly** (e.g. auth → concrete HTTP client). That is acceptable for MVP; refactors should move toward ports without changing the public API.

| Concern   | Current                                     | Target (incremental)                                                                         |
| --------- | ------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Ports     | [qpub/interfaces.go](../qpub/interfaces.go) | Add `internal/port/http.go`, `internal/port/websocket.go`; application depends on interfaces |
| Bootstrap | Inline in `qpub/*.go`                       | Optional `internal/bootstrap/socket.go`, `rest.go` if wiring grows                           |
| Event bus | `internal/emitter`                          | Keep internal; expose typed callbacks on managers/connection                                 |
| Logger    | `internal/logger`                           | Optional `port.Logger` for custom sinks                                                      |

No big-bang rename required: new code follows import rules; existing code is tightened when touched.

## Adding a feature (checklist)

1. Wire/protocol change → `protocol/` (+ tests).
2. Config → `option/` (+ defaults documented in code).
3. Use case → appropriate application package (`auth`, `channel`, `connection`, `queue`).
4. I/O → `transport/` or `internal/`.
5. Export types/constants → `qpub/doc.go` or `qpub/interfaces.go`.
6. Wire in `qpub/NewSocket` or `NewRest`.
7. Update [implementation-status.md](./implementation-status.md) when parity phase completes.

## Related

- [implementation-status.md](./implementation-status.md) — plan phases and completion state
- [cross-sdk-api.md](./cross-sdk-api.md) — JS vs Go API
