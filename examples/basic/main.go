// Basic REST publish example (set QPUB_API_KEY).
package main

import (
	"context"
	"log"
	"os"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/qpub"
)

func main() {
	key := os.Getenv("QPUB_API_KEY")
	if key == "" {
		log.Fatal("QPUB_API_KEY required")
	}
	rest := qpub.NewRest(qpub.WithAPIKey(key))
	defer rest.Reset()

	_, err := rest.Channels.Get("my-channel").Publish(context.Background(), "Hello from Go!", channel.PublishOptions{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("published")
}
