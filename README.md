# QPub Go SDK

Official Go client for [QPub](https://qpub.io) real-time messaging (Channels and Queues).

**Stability:** v0.x — the public API may change before v1.0.0. Pin a release, for example:

```bash
go get github.com/qpubio/qpub-go@v0.1.0
```

Import path for application code:

```bash
go get github.com/qpubio/qpub-go/qpub@v0.1.0
```

See [CHANGELOG.md](CHANGELOG.md) for release notes.

## Install (latest pseudo-version)

```bash
go get github.com/qpubio/qpub-go/qpub
```

## Quick start

### Socket (subscribe / publish)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/qpubio/qpub-go/channel"
    "github.com/qpubio/qpub-go/qpub"
)

func main() {
    socket := qpub.NewSocket(qpub.WithAPIKey("YOUR_PUBLIC_ID:YOUR_SECRET"))
    defer socket.Reset()

    ch := socket.Channels.Get("my-channel")
    err := ch.Subscribe(context.Background(), func(m qpub.Message) {
        fmt.Println("received", string(m.Data))
    }, channel.SubscribeOptions{})
    if err != nil {
        log.Fatal(err)
    }

    _ = ch.Publish(context.Background(), "Hello!", channel.PublishOptions{})
}
```

### REST (publish / queues)

```go
import (
    "context"
    "github.com/qpubio/qpub-go/channel"
    "github.com/qpubio/qpub-go/qpub"
)

rest := qpub.NewRest(qpub.WithAPIKey("YOUR_PUBLIC_ID:YOUR_SECRET"))
defer rest.Reset()

ch := rest.Channels.Get("my-channel")
_, err := ch.Publish(context.Background(), "Hello!", channel.PublishOptions{})
```

See [`examples/`](examples/) and the [docs/](docs/) folder:

- [Architecture & layer rules](docs/architecture.md)
- [Implementation status (plan phases)](docs/implementation-status.md)
- [Documentation index](docs/README.md)
- [Contributing & development](CONTRIBUTING.md)

## License

Apache-2.0
