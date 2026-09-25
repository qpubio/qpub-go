// Queue worker example (requires QPUB_API_KEY).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go"
)

func main() {
	key := os.Getenv("QPUB_API_KEY")
	queueName := os.Getenv("QPUB_QUEUE")
	if key == "" || queueName == "" {
		log.Fatal("QPUB_API_KEY and QPUB_QUEUE required")
	}

	rest := qpub.NewRest(qpub.WithAPIKey(key))
	defer rest.Reset()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := rest.Queues.RunWorker(ctx, queueName, func(ctx context.Context, job protocol.QueueJob) (interface{}, error) {
		fmt.Println("processing job", job.ID, string(job.Payload))
		result := map[string]string{"status": "ok"}
		fmt.Println("done job", job.ID, result)
		return result, nil
	}, protocol.RunWorkerOptions{})
	if err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
