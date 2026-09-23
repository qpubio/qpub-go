# qpub-go implementation status

Single source of truth for **parity with qpub-js v2.1.0** and the implementation plan. Update this file when a phase or item changes state. Spec reference: [qpub-js](https://github.com/qpubio/qpub-js) at v2.1.0.

**Legend:** Done | Partial | Todo

---

## Phase 0 — Foundation and contract tests

| Item                                         | State   | Notes                                                               |
| -------------------------------------------- | ------- | ------------------------------------------------------------------- |
| Module `github.com/qpubio/qpub-go`, Go 1.22+ | Done    | [go.mod](../go.mod)                                                 |
| Apache-2.0                                   | Done    | [LICENSE](../LICENSE)                                               |
| `protocol/` actions and messages             | Done    | [protocol/](../protocol/)                                           |
| ApiKey parse                                 | Done    | [internal/apikey](../internal/apikey/)                              |
| JWT HS256 sign/decode/expiry                 | Done    | [internal/jwt](../internal/jwt/)                                    |
| HMAC + canonical token request string        | Done    | [auth/manager.go](../auth/manager.go) `BuildCanonicalString`        |
| Auth unit tests (subset of qpub-js)          | Partial | [auth/manager_test.go](../auth/manager_test.go) — not full JS suite |
| Golden vectors shared with qpub-js           | Todo    | Cross-language test fixture file optional                           |

---

## Phase 1 — Options, HTTP, AuthManager

| Item                                                      | State | Notes                                   |
| --------------------------------------------------------- | ----- | --------------------------------------- |
| `option/` defaults (hosts, reconnect, ping, auth retries) | Done  | [option/option.go](../option/option.go) |
| REST base URL builder                                     | Done  | `option.BuildRestBaseURL`               |
| AuthManager: Authenticate, tokens, headers, query URL     | Done  | [auth/manager.go](../auth/manager.go)   |
| Auth events (token updated/expired/error)                 | Done  | [events/events.go](../events/events.go) |
| `NewRest`, Reset, GetInstanceID                           | Done  | [qpub/rest.go](../qpub/rest.go)         |

---

## Phase 2 — REST channels

| Item                         | State | Notes                                           |
| ---------------------------- | ----- | ----------------------------------------------- |
| Per-channel publish          | Done  | [channel/rest.go](../channel/rest.go)           |
| PublishBatch                 | Done  | `RestManager.PublishBatch`                      |
| Manager Get/Has/Remove/Reset | Done  |                                                 |
| httptest integration         | Done  | [channel/rest_test.go](../channel/rest_test.go) |

---

## Phase 3 — REST queues

| Item                                          | State   | Notes                                                          |
| --------------------------------------------- | ------- | -------------------------------------------------------------- |
| Enqueue, GetJob, ListJobs, Cancel, Retry      | Done    | [queue/manager.go](../queue/manager.go)                        |
| GetConfig, UpdateConfig                       | Done    |                                                                |
| RegisterWorker, Heartbeat, Pull, Ack, Nack    | Done    |                                                                |
| RunWorker / StopWorker (context + loop)       | Done    |                                                                |
| Worker integration test (pull → ack sequence) | Partial | [queue/manager_test.go](../queue/manager_test.go) enqueue only |

---

## Phase 4 — WebSocket transport and connection

| Item                              | State   | Notes                                                   |
| --------------------------------- | ------- | ------------------------------------------------------- |
| WebSocket client wrapper          | Done    | [transport/ws/client.go](../transport/ws/client.go)     |
| Connect with auth URL             | Done    | [connection/connection.go](../connection/connection.go) |
| Connection lifecycle events       | Done    |                                                         |
| Auto-connect                      | Done    | [qpub/socket.go](../qpub/socket.go)                     |
| Auto-reconnect + backoff          | Partial | Basic reconnect; review parity with qpub-js edge cases  |
| Auto-authenticate on connect      | Done    |                                                         |
| Ping/pong RTT (`id` correlation)  | Partial | Implemented; needs unit tests                           |
| Resubscribe after reconnect       | Done    | Calls channel manager                                   |
| Connection unit/integration tests | Todo    | Port scenarios from qpub-js `connection.test.ts`        |

---

## Phase 5 — Socket channels

| Item                                     | State | Notes                                                     |
| ---------------------------------------- | ----- | --------------------------------------------------------- |
| Subscribe / unsubscribe / publish wire   | Done  | [channel/socket.go](../channel/socket.go)                 |
| MESSAGE dispatch to handler              | Done  |                                                           |
| Pause / resume / buffer                  | Done  |                                                           |
| Manager ref-count + Release              | Done  | [channel/socket.go](../channel/socket.go) `SocketManager` |
| Pending subscribe + resubscribe all      | Done  |                                                           |
| Operation queue during pending sub/unsub | Todo  | Simplified vs qpub-js `socket-channel.ts`                 |
| SUBSCRIBED / UNSUBSCRIBED handling       | Todo  |                                                           |
| Subscribe by `event` filter              | Todo  |                                                           |
| Channel unit tests                       | Todo  | Port qpub-js socket-channel tests                         |

---

## Phase 6 — Socket facade and reset

| Item                                                 | State   | Notes                                    |
| ---------------------------------------------------- | ------- | ---------------------------------------- |
| `NewSocket`, GetInstanceID                           | Done    | [qpub/socket.go](../qpub/socket.go)      |
| Reset order (connection → channels → auth → options) | Done    |                                          |
| Thread-safety documentation                          | Partial | See [architecture.md](./architecture.md) |

---

## Phase 7 — Exports and testing package

| Item                                            | State   | Notes                                                                              |
| ----------------------------------------------- | ------- | ---------------------------------------------------------------------------------- |
| Exported consumer interfaces                    | Partial | [qpub/interfaces.go](../qpub/interfaces.go)                                        |
| Event constants re-export                       | Done    | [qpub/doc.go](../qpub/doc.go)                                                      |
| `testing/` mocks and helpers                    | Partial | [testing/testing.go](../testing/testing.go) — MockHTTP, NewTestRest, NewTestSocket |
| Parity with qpub-js TestContainer / MockFactory | Todo    | Go uses interfaces + manual mocks                                                  |
| Runtime DI container                            | N/A     | By design: composition in `qpub/` per [architecture.md](./architecture.md)         |

---

## Phase 8 — Examples, CI, release, docs

| Item                                                    | State | Notes                                                   |
| ------------------------------------------------------- | ----- | ------------------------------------------------------- |
| README quick start                                      | Done  | [README.md](../README.md)                               |
| cross-sdk-api.md                                        | Done  | [cross-sdk-api.md](./cross-sdk-api.md)                  |
| architecture.md                                         | Done  | [architecture.md](./architecture.md)                    |
| Example: basic REST publish                             | Done  | [examples/basic](../examples/basic/)                    |
| Example: socket pub/sub                                 | Todo  |                                                         |
| Example: token auth (CreateTokenRequest + RequestToken) | Todo  |                                                         |
| Example: queue worker                                   | Todo  |                                                         |
| CI: `go test -race ./...`                               | Done  | [.github/workflows/ci.yml](../.github/workflows/ci.yml) |
| CI: golangci-lint                                       | Todo  |                                                         |
| Release workflow / semver tags                          | Todo  |                                                         |
| qpub.io / shared docs quickstart (Go)                   | Todo  | After live SDK verification                             |

---

## Architecture alignment (ongoing)

| Item                                              | State   | Notes                                                     |
| ------------------------------------------------- | ------- | --------------------------------------------------------- |
| Layer import rules documented                     | Done    | [architecture.md](./architecture.md)                      |
| Application packages use HTTP port interface only | Partial | `auth.HTTPDoer` exists; queue/channel use concrete client |
| Optional `internal/bootstrap` extraction          | Todo    | When wiring grows                                         |
| `internal/port` for HTTP/WS/Logger                | Todo    | Incremental                                               |

---

## Suggested priority (remaining work)

1. Socket **tests** + **SUBSCRIBED/operation queue** (phases 4–5 hardening).
2. **Examples** for socket, token auth, queue worker (phase 8).
3. **Auth golden tests** + more queue worker tests (phase 0/3).
4. **Lint + release** CI (phase 8).
5. **Port interfaces** refactor when touching auth/transport (architecture target).

---

## Out of scope (by plan)

- React / browser bundle / UMD
- Backend-style DDD folder tree
- Feature parity with `@qpub/sdk/react`
