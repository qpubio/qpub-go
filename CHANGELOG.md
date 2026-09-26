# Changelog

All notable changes to the QPub Go SDK will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-09-26

### Changed

- Socket and REST client instance IDs use UUIDv7 (`internal/instanceid`), aligned with qpub-js.

### Added

- Broader unit and integration tests (auth, connection, socket channel/manager, instance ID, `testing/` helpers).

## [0.2.0] - 2026-09-25

### Changed

- Import the client as `github.com/qpubio/qpub-go` (facade at module root; no `/qpub` suffix).

## [0.1.0] - 2026-09-25

### Added

- First public module: `github.com/qpubio/qpub-go`
- **Socket** client (`qpub.NewSocket`) — WebSocket pub/sub, reconnect, ping, token/API key auth
- **Rest** client (`qpub.NewRest`) — HTTP channel publish and batch publish
- **Queues** — enqueue, job management, worker pull/ack/nack, `RunWorker`
- **Auth** — `GenerateToken`, `IssueToken`, `CreateTokenRequest`, `RequestToken`
- Package `testing/` — `MockHTTP`, `MockWS`, `NewTestRest`, `NewTestSocket`
- Examples under `examples/` (basic, socket, token-auth, queue-worker)
- CI (`go test -race`) and tag-driven release workflow

### Notes

- **v0.x** — public API may change before v1.0.0; pin with `go get github.com/qpubio/qpub-go@v0.1.0`
- Cross-SDK parity details: [docs/implementation-status.md](docs/implementation-status.md)
- Server-side / backend use; no browser bundle or React helpers

[0.3.0]: https://github.com/qpubio/qpub-go/releases/tag/v0.3.0
[0.2.0]: https://github.com/qpubio/qpub-go/releases/tag/v0.2.0
[0.1.0]: https://github.com/qpubio/qpub-go/releases/tag/v0.1.0
