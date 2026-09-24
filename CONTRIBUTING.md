# Contributing to qpub-go

Thank you for contributing to the QPub Go SDK. This guide covers local development, running examples, testing, linting, and debugging.

## Prerequisites

- **Go** 1.22 or newer (see [go.mod](go.mod) for the version used in CI)
- **Git**
- **golangci-lint** (optional locally; CI runs it via [.github/workflows/ci.yml](.github/workflows/ci.yml))
- **Delve** (`dlv`) optional, for debugging tests

## Clone and setup

```bash
git clone git@github.com:qpubio/qpub-go.git
cd qpub-go
go mod download
```

### Repository layout

Packages are organized by layer (API, application, protocol, transport). Before moving code or adding features, read [docs/architecture.md](docs/architecture.md) for import rules and package roles.

Track parity and roadmap items in [docs/implementation-status.md](docs/implementation-status.md) only—update that file when you complete or change plan phases.

## Run examples

Examples live under [examples/](examples/). Use environment variables for credentials; never commit API keys.

| Example | Command | Environment |
|---------|---------|-------------|
| REST publish | `go run ./examples/basic` | `QPUB_API_KEY=publicId:secret` |
| Socket subscribe | `go run ./examples/socket` | `QPUB_API_KEY=publicId:secret` |
| Token auth flow | `go run ./examples/token-auth` | `QPUB_API_KEY=publicId:secret` |
| Queue worker | `go run ./examples/queue-worker` | `QPUB_API_KEY`, `QPUB_QUEUE` |

### Custom endpoints or options

Use functional options when constructing clients:

```go
import (
    "github.com/qpubio/qpub-go/option"
    "github.com/qpubio/qpub-go/qpub"
)

rest := qpub.NewRest(
    qpub.WithAPIKey(os.Getenv("QPUB_API_KEY")),
    option.WithIsSecure(false),
    func(o *option.Option) {
        o.HTTPHost = "localhost"
        p := 8080
        o.HTTPPort = &p
    },
)
```

For many fields at once, start from defaults and merge:

```go
opts := option.DefaultOption()
opts.Debug = true
opts.LogLevel = "debug"
opts.APIKey = os.Getenv("QPUB_API_KEY")

socket := qpub.NewSocket(option.FromOption(opts), option.WithAutoConnect(true))
```

## Test

Run the full suite:

```bash
go test ./...
```

Match CI (race detector):

```bash
go test -race ./...
```

Focused packages:

```bash
go test ./auth/... -v
go test ./channel/... -v
go test ./connection/... -v
go test ./queue/... -v
```

Canonical signing golden tests read fixtures from [auth/testdata/](auth/testdata/).

Use the [testing/](testing/) package helpers (`MockHTTP`, `NewTestRest`, `NewTestSocket`) when building tests around the public API.

## Lint

Local lint (install [golangci-lint](https://golangci-lint.run/) if needed):

```bash
golangci-lint run
```

Configuration: [.golangci.yml](.golangci.yml). CI runs the same linters on push and pull requests.

## Debug

### SDK logging

Enable verbose internal logs via options (see [option/option.go](option/option.go)):

```go
opts := option.DefaultOption()
opts.Debug = true
opts.LogLevel = "trace" // error | warn | info | debug | trace
opts.APIKey = os.Getenv("QPUB_API_KEY")

client := qpub.NewRest(option.FromOption(opts))
```

Logs go to the standard library `log` output with component tags.

### Delve (tests)

```bash
dlv test ./channel -- -test.run TestSocketSubscribeWaitsForSubscribed
```

### Cloud debugging

When running examples against QPub Cloud:

- Pass secrets only through the environment or a local untracked `.env` (not committed).
- Use debug log level temporarily; redact tokens in issue reports.

## Contributing workflow

1. Branch from `main` or `dev` (match the branch your PR targets): `feature/…` or `fix/…`.
2. Follow [docs/architecture.md](docs/architecture.md) import boundaries.
3. Add or update tests for behavior changes.
4. Run `go test -race ./...` and `golangci-lint run` before opening a PR.
5. If the change affects parity or plan phases, update [docs/implementation-status.md](docs/implementation-status.md).
6. Open a PR with a clear description and test plan.

## Further reading

- [docs/README.md](docs/README.md) — documentation index
- [docs/architecture.md](docs/architecture.md) — layers and dependencies
- [docs/implementation-status.md](docs/implementation-status.md) — feature parity checklist and estimates

## License

Contributions are licensed under Apache-2.0, same as the project ([LICENSE](LICENSE)).
