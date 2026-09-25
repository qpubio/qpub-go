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
| `channel.on("subscribed", fn)` | `ch.On(qpub.ChannelEvents.Subscribed, fn)` |
| `channel.subscribe(fn, { event })` | `ch.Subscribe(ctx, fn, channel.SubscribeOptions{Event: "..."})` |
| `rest.queues.enqueue(...)` | `rest.Queues.Enqueue(ctx, ...)` |

## Testing (Go)

Import `github.com/qpubio/qpub-go/testing` for `MockHTTP`, `NewTestRest`, and `NewTestSocket` (see [CONTRIBUTING.md](../CONTRIBUTING.md)).

## Go adaptations

- **Async**: use `context.Context` and `(T, error)` instead of Promises.
- **Events**: `On(event, fn)` on connection, auth, and socket channels instead of `EventEmitter.on`.
- **Connect**: `Connection.WaitUntilConnected(ctx)` before subscribe when not using callbacks on `connected`.
- **Options**: functional helpers (`WithAPIKey`, `WithAutoConnect`) or `option.FromOption` after `DefaultOption()`.
- **React**: `@qpub/sdk/react` has no Go equivalent.

## Token authentication

Same three server-side flows: `GenerateToken`, `IssueToken`, `CreateTokenRequest`, plus client `RequestToken`. Canonical token-request signing follows the shared cross-SDK contract (see [implementation-status.md](./implementation-status.md)).
