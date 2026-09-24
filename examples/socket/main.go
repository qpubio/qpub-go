// Socket subscribe example (requires QPUB_API_KEY).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/qpub"
)

func main() {
	key := os.Getenv("QPUB_API_KEY")
	if key == "" {
		log.Fatal("QPUB_API_KEY required")
	}

	socket := qpub.NewSocket(qpub.WithAPIKey(key))
	defer socket.Reset()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := socket.Connection.WaitUntilConnected(ctx); err != nil {
		log.Fatal(err)
	}

	ch := socket.Channels.Get("my-channel")
	err := ch.Subscribe(ctx, func(m qpub.Message) {
		fmt.Println("message:", string(m.Data))
	}, channel.SubscribeOptions{Timeout: 30 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
}
