# qpub-go implementation status

Single source of truth for **parity with qpub-js v2.1.0** and the implementation plan. Update this file when a phase or item changes state. Spec reference: [qpub-js](https://github.com/qpubio/qpub-js) at v2.1.0.

**Legend:** Done | Partial | Todo

---

## Phase 0 — Foundation and contract tests

| Item                                         | State | Notes                                                                 |
| -------------------------------------------- | ----- | --------------------------------------------------------------------- |
| Module `github.com/qpubio/qpub-go`, Go 1.22+ | Done  | [go.mod](../go.mod)                                                   |
| Apache-2.0                                   | Done  | [LICENSE](../LICENSE)                                                 |
| `protocol/` actions and messages             | Done  | [protocol/](../protocol/)                                             |
| ApiKey parse                                 | Done  | [internal/apikey](../internal/apikey/)                                |
| JWT HS256 sign/decode/expiry                 | Done  | [internal/jwt](../internal/jwt/)                                      |
| HMAC + canonical token request string        | Done  | [auth/manager.go](../auth/manager.go) `BuildCanonicalString`          |
| Auth unit tests (subset of qpub-js)          | Done  | [auth/manager_test.go](../auth/manager_test.go)                       |
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

| Item                                          | State | Notes                                                          |
| --------------------------------------------- | ----- | -------------------------------------------------------------- |
| Enqueue, GetJob, ListJobs, Cancel, Retry      | Done  | [queue/manager.go](../queue/manager.go)                        |
| GetConfig, UpdateConfig                       | Done  |                                                                |
| RegisterWorker, Heartbeat, Pull, Ack, Nack    | Done  |                                                                |
| RunWorker / StopWorker (context + loop)       | Done  |                                                                |
| Worker integration test (pull → ack sequence) | Done  | [queue/manager_test.go](../queue/manager_test.go) `TestPullAndAck` |

---

## Phase 4 — WebSocket transport and connection

| Item                              | State   | Notes                                                                 |
| --------------------------------- | ------- | --------------------------------------------------------------------- |
| WebSocket client wrapper          | Done    | [transport/ws/client.go](../transport/ws/client.go)                   |
| Connect with auth URL             | Done    | [connection/connection.go](../connection/connection.go)               |
| Connection lifecycle events       | Done    |                                                                       |
| Auto-connect                      | Done    | [qpub/socket.go](../qpub/socket.go)                                   |
| Auto-reconnect + backoff          | Partial | Basic reconnect; edge cases vs full qpub-js not exhaustively tested     |
| Auto-authenticate on connect      | Done    |                                                                       |
| Ping/pong RTT (`id` correlation)  | Done    | [connection/ping_test.go](../connection/ping_test.go)                 |
| Resubscribe after reconnect       | Done    | Calls channel manager                                                 |
| Connection unit/integration tests | Partial | Ping RTT unit test; full WS integration tests still Todo              |

---

## Phase 5 — Socket channels

| Item                                     | State   | Notes                                                                 |
| ---------------------------------------- | ------- | --------------------------------------------------------------------- |
| Subscribe / unsubscribe / publish wire   | Done    | [channel/socket.go](../channel/socket.go)                             |
| MESSAGE dispatch to handler              | Done    |                                                                       |
| Pause / resume / buffer                  | Done    | [channel/socket_test.go](../channel/socket_test.go)                   |
| Manager ref-count + Release              | Done    | `SocketManager`                                                       |
| Pending subscribe + resubscribe all      | Done    |                                                                       |
| Operation queue during pending sub/unsub | Done    | Queued ops when pending subscribe/unsubscribe                         |
| SUBSCRIBED / UNSUBSCRIBED handling       | Done    | `HandleIncoming`; subscribe/unsubscribe wait for ack                  |
| Subscribe by `event` filter              | Done    | `SubscribeOptions.Event`                                              |
| Channel unit tests                       | Done    | [channel/socket_test.go](../channel/socket_test.go)                   |

---

## Phase 6 — Socket facade and reset

| Item                                                 | State | Notes                                    |
| ---------------------------------------------------- | ----- | ---------------------------------------- |
| `NewSocket`, GetInstanceID                           | Done  | [qpub/socket.go](../qpub/socket.go)      |
| Reset order (connection → channels → auth → options) | Done  |                                          |
| Thread-safety documentation                          | Done  | [architecture.md](./architecture.md) + serial handlers per channel |

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
| Example: socket pub/sub                                 | Done  | [examples/socket](../examples/socket/)                  |
| Example: token auth (CreateTokenRequest + RequestToken) | Done  | [examples/token-auth](../examples/token-auth/)          |
| Example: queue worker                                   | Done  | [examples/queue-worker](../examples/queue-worker/)      |
| CI: `go test -race ./...`                               | Done  | [.github/workflows/ci.yml](../.github/workflows/ci.yml) |
| CI: golangci-lint                                       | Done  | [.golangci.yml](../.golangci.yml)                       |
| Release workflow / semver tags                          | Todo  |                                                         |
| qpub.io / shared docs quickstart (Go)                   | Todo  | After live SDK verification                             |

---

## Architecture alignment (ongoing)

| Item                                              | State   | Notes                                                                 |
| ------------------------------------------------- | ------- | --------------------------------------------------------------------- |
| Layer import rules documented                     | Done    | [architecture.md](./architecture.md)                                  |
| Application packages use HTTP port interface only | Partial | [internal/port/http.go](../internal/port/http.go) added; wire queue/channel/auth incrementally |
| Optional `internal/bootstrap` extraction          | Todo    | When wiring grows                                                     |
| `MessageSender` port for WebSocket send           | Done    | [channel/ws_sender.go](../channel/ws_sender.go)                       |

---

## Suggested priority (remaining work)

1. **Connection** integration tests with real WebSocket test server (phase 4).
2. **Expand testing/** mocks (WS, logger) and exported interface coverage (phase 7).
3. **Release** workflow and semver tags (phase 8).
4. **Wire** `internal/port.HTTPClient` through queue/channel (architecture).
5. Live verification → shared docs quickstart (phase 8).

---

## Out of scope (by plan)

- React / browser bundle / UMD
- Backend-style DDD folder tree
- Feature parity with `@qpub/sdk/react`
