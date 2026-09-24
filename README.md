# QPub Go SDK

Official Go client for [QPub](https://qpub.io) real-time messaging (Channels and Queues).

## Install

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
