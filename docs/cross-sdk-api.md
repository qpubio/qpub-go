# Cross-SDK API (JavaScript vs Go)

Wire protocol and REST paths are identical across SDKs. Client ergonomics follow each language.

## Entry points

| JavaScript | Go |
|------------|-----|
| `new QPub.Socket({ apiKey })` | `qpub.NewSocket(qpub.WithAPIKey("id:secret"))` |
| `new QPub.Rest({ apiKey })` | `qpub.NewRest(qpub.WithAPIKey("id:secret"))` |

## Composition

| JavaScript | Go |
|------------|-----|
| `socket.auth` | `socket.Auth` |
| `socket.connection` | `socket.Connection` |
| `socket.channels.get(name)` | `socket.Channels.Get(name)` |
| `rest.queues.enqueue(...)` | `rest.Queues.Enqueue(ctx, ...)` |

## Go adaptations

- **Async**: use `context.Context` and `(T, error)` instead of Promises.
- **Events**: `On(event, fn)` on connection/auth instead of `EventEmitter.on`.
- **Options**: functional helpers (`WithAPIKey`, `WithAutoConnect`) or `option.FromOption` after `DefaultOption()`.
- **React**: `@qpub/sdk/react` has no Go equivalent.

## Token authentication

Same three server-side flows: `GenerateToken`, `IssueToken`, `CreateTokenRequest`, plus client `RequestToken`. Canonical signing matches qpub-js 2.0.9+.
