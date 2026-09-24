# qpub-go implementation status

Single source of truth for **parity with qpub-js v2.1.0** and the implementation plan. Update this file when a phase or item changes state. Spec reference: [qpub-js](https://github.com/qpubio/qpub-js) at v2.1.0.

**Legend:** Done | Partial | Todo

---

## Parity estimate (qpub-js v2.1.0 core)

**Last reviewed:** 2026-09-24 (post live verification + parity pass)

Excludes React, UMD, and runtime DI (out of scope). Percentages are approximate.

| Dimension                                                 | Approx. | Notes                                                                                                        |
| --------------------------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------ |
| Public API surface (Socket, Rest, auth, channels, queues) | ~92%    | Channel `On`, multi-event subscribe, `WaitUntilConnected` on connection interface                            |
| Runtime behavior (reconnect, socket edge cases)           | ~85%    | Live smoke OK; reconnect/ping tests added; Node-only server ping watchdog N/A for gorilla client               |
| Automated test parity                                     | ~65%    | WS httptest integration, socket manager tests, protocol wire tests; not every qpub-js test line ported       |
| DevEx / shipping                                          | ~80%    | Release workflow, [CONTRIBUTING.md](../CONTRIBUTING.md), `testing/` MockWS; shared docs Go tabs not done yet |

**Practical summary**

- **~85–90%** — suitable for Go backends (REST, queues, socket pub/sub, token auth).
- **~80%** — behavioral equivalence under reconnect and multi-event subscribe scenarios covered by tests.

Contributor guide: [CONTRIBUTING.md](../CONTRIBUTING.md).

---

## Live verification

| Example | Status | Notes |
| ------- | ------ | ----- |
| `examples/basic` | Done | REST publish |
| `examples/socket` | Done | Subscribe + MESSAGE delivery (string ULID `id` on wire) |
| `examples/token-auth` | Done | Token request flow |
| `examples/queue-worker` | Done | Pull → process → ack |

---

## Parity test checklist (qpub-js → qpub-go)

| qpub-js test file | Go coverage |
| ----------------- | ----------- |
| `__tests__/unit/auth-manager.test.ts` | [auth/manager_test.go](../auth/manager_test.go), canonical vectors |
| `__tests__/unit/connection.test.ts` | [connection/handle_message_test.go](../connection/handle_message_test.go), [connection/reconnect_test.go](../connection/reconnect_test.go), [connection/ws_integration_test.go](../connection/ws_integration_test.go), [connection/ping_test.go](../connection/ping_test.go) |
| `__tests__/unit/socket-channel.test.ts` | [channel/socket_test.go](../channel/socket_test.go), [channel/socket_multi_event_test.go](../channel/socket_multi_event_test.go) |
| `__tests__/unit/socket-channel-manager.test.ts` | [channel/socket_manager_test.go](../channel/socket_manager_test.go) |
| `__tests__/integration/auth-connection-channel.test.ts` | Partial — composition via unit tests + WS integration |

---

## Phase 0 — Foundation and contract tests

| Item                                         | State | Notes                                                                                                                                |
| -------------------------------------------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Module `github.com/qpubio/qpub-go`, Go 1.22+ | Done  | [go.mod](../go.mod)                                                                                                                  |
| Apache-2.0                                   | Done  | [LICENSE](../LICENSE)                                                                                                                |
| `protocol/` actions and messages             | Done  | [protocol/](../protocol/), [protocol/wire_test.go](../protocol/wire_test.go) (`status_code`, server shapes)                         |
| ApiKey parse                                 | Done  | [internal/apikey](../internal/apikey/)                                                                                               |
| JWT HS256 sign/decode/expiry                 | Done  | [internal/jwt](../internal/jwt/)                                                                                                     |
| HMAC + canonical token request string        | Done  | [auth/manager.go](../auth/manager.go) `BuildCanonicalString`                                                                         |
| Auth unit tests (subset of qpub-js)          | Done  | [auth/manager_test.go](../auth/manager_test.go)                                                                                      |
| Golden vectors shared with qpub-js           | Done  | [auth/testdata/canonical_vectors.json](../auth/testdata/canonical_vectors.json), [auth/canonical_test.go](../auth/canonical_test.go) |

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

| Item                                          | State | Notes                                                              |
| --------------------------------------------- | ----- | ------------------------------------------------------------------ |
| Enqueue, GetJob, ListJobs, Cancel, Retry      | Done  | [queue/manager.go](../queue/manager.go)                            |
| GetConfig, UpdateConfig                       | Done  |                                                                    |
| RegisterWorker, Heartbeat, Pull, Ack, Nack    | Done  |                                                                    |
| RunWorker / StopWorker (context + loop)       | Done  |                                                                    |
| Worker integration test (pull → ack sequence) | Done  | [queue/manager_test.go](../queue/manager_test.go) `TestPullAndAck` |

---

## Phase 4 — WebSocket transport and connection

| Item                              | State | Notes                                                                                               |
| --------------------------------- | ----- | --------------------------------------------------------------------------------------------------- |
| WebSocket client wrapper          | Done  | [transport/ws/client.go](../transport/ws/client.go)                                                 |
| Connect with auth URL             | Done  | [connection/connection.go](../connection/connection.go)                                             |
| Connection lifecycle events       | Done  |                                                                                                     |
| Auto-connect                      | Done  | [qpub/socket.go](../qpub/socket.go)                                                                 |
| Auto-reconnect + backoff          | Done  | [connection/reconnect_test.go](../connection/reconnect_test.go)                                     |
| Auto-authenticate on connect      | Done  |                                                                                                     |
| Ping/pong RTT (`id` correlation)  | Done  | [connection/ping_test.go](../connection/ping_test.go)                                               |
| Resubscribe after reconnect       | Done  | Calls channel manager                                                                               |
| MESSAGE routing (string message id) | Done | Peek `action` before unmarshaling ping `id` as int                                                  |
| Malformed JSON → failed + context | Done  | `message_processing` context on [events.ConnectionFailed](../events/events.go)                      |
| Connection unit/integration tests | Done  | [connection/ws_integration_test.go](../connection/ws_integration_test.go)                           |
| `WaitUntilConnected`              | Done  | On `*Conn` and [Connection interface](../qpub/interfaces.go)                                        |

---

## Phase 5 — Socket channels

| Item                                     | State | Notes                                                                                      |
| ---------------------------------------- | ----- | ------------------------------------------------------------------------------------------ |
| Subscribe / unsubscribe / publish wire   | Done  | [channel/socket.go](../channel/socket.go)                                                  |
| MESSAGE dispatch to handler              | Done  |                                                                                            |
| Multi event-specific subscriptions       | Done  | No extra SUBSCRIBE when channel already subscribed                                         |
| Subscribe when disconnected              | Done  | Returns error (matches qpub-js)                                                            |
| Catch-all update when already subscribed | Done  | Updates handler without redundant wire                                                     |
| Channel lifecycle `On(event, fn)`        | Done  |                                                                                            |
| Event-specific unsubscribe               | Done  | [UnsubscribeOptions](../channel/socket.go)                                                 |
| Pause / resume / buffer                  | Done  | [channel/socket_test.go](../channel/socket_test.go)                                         |
| Manager ref-count + Release              | Done  | [channel/socket_manager_test.go](../channel/socket_manager_test.go)                        |
| Pending subscribe + resubscribe all      | Done  |                                                                                            |
| Operation queue during pending sub/unsub | Done  |                                                                                            |
| SUBSCRIBED / UNSUBSCRIBED handling       | Done  |                                                                                            |
| Channel unit tests                       | Done  |                                                                                            |

---

## Phase 6 — Socket facade and reset

| Item                                                 | State | Notes                                                              |
| ---------------------------------------------------- | ----- | ------------------------------------------------------------------ |
| `NewSocket`, GetInstanceID                           | Done  | [qpub/socket.go](../qpub/socket.go)                                |
| Reset order (connection → channels → auth → options) | Done  |                                                                    |
| Thread-safety documentation                          | Done  | [architecture.md](./architecture.md) + serial handlers per channel |

---

## Phase 7 — Exports and testing package

| Item                                            | State   | Notes                                                                              |
| ----------------------------------------------- | ------- | ---------------------------------------------------------------------------------- |
| Exported consumer interfaces                    | Done    | [qpub/interfaces.go](../qpub/interfaces.go)                                        |
| Event constants re-export                       | Done    | [qpub/doc.go](../qpub/doc.go)                                                      |
| `testing/` mocks and helpers                    | Done    | MockHTTP, MockWS, NewTestRest, NewTestSocket — [testing/](../testing/)           |
| Parity with qpub-js TestContainer / MockFactory | Partial | Subset via MockHTTP/MockWS; no DI container (by design)                          |
| Runtime DI container                            | N/A     | By design: composition in `qpub/` per [architecture.md](./architecture.md)         |

---

## Phase 8 — Examples, CI, release, docs

| Item                                                    | State | Notes                                                   |
| ------------------------------------------------------- | ----- | ------------------------------------------------------- |
| README quick start                                      | Done  | [README.md](../README.md)                               |
| cross-sdk-api.md                                        | Done  | [cross-sdk-api.md](./cross-sdk-api.md)                  |
| architecture.md                                         | Done  | [architecture.md](./architecture.md)                    |
| Example: basic REST publish                             | Done  | [examples/basic](../examples/basic/)                    |
| Example: socket pub/sub                                 | Done  | [examples/socket](../examples/socket/)                  |
| Example: token auth (CreateTokenRequest + RequestToken) | Done  | [examples/token-auth](../examples/token-auth/)          |
| Example: queue worker                                   | Done  | [examples/queue-worker](../examples/queue-worker/)      |
| CI: `go test -race ./...`                               | Done  | [.github/workflows/ci.yml](../.github/workflows/ci.yml) |
| CI: golangci-lint                                       | Done  | [.golangci.yml](../.golangci.yml)                       |
| Contributor develop/test/debug guide                    | Done  | [CONTRIBUTING.md](../CONTRIBUTING.md)                   |
| Release workflow / semver tags                          | Done  | [.github/workflows/release.yml](../.github/workflows/release.yml) |
| qpub.io / shared docs (Go examples on existing pages)   | Todo  | Add Go tabs/snippets on existing SDK pages when ready — no separate language page |

---

## Architecture alignment (ongoing)

| Item                                              | State   | Notes                                                                                          |
| ------------------------------------------------- | ------- | ---------------------------------------------------------------------------------------------- |
| Layer import rules documented                     | Done    | [architecture.md](./architecture.md)                                                           |
| Application packages use HTTP port interface only | Partial | [internal/port/http.go](../internal/port/http.go) added; wire queue/channel/auth incrementally |
| Optional `internal/bootstrap` extraction          | Todo    | When wiring grows                                                                              |
| `MessageSender` port for WebSocket send           | Done    | [channel/ws_sender.go](../channel/ws_sender.go) (+ `IsConnected`)                              |

---

## Out of scope (by plan)

- React / browser bundle / UMD
- Backend-style DDD folder tree
- Feature parity with `@qpub/sdk/react`
- qpub-js `ServiceContainer` / runtime DI
