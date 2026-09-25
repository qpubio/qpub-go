// Token auth: server-side CreateTokenRequest, client-side RequestToken.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go"
)

func main() {
	key := os.Getenv("QPUB_API_KEY")
	if key == "" {
		log.Fatal("QPUB_API_KEY required (publicId:secret)")
	}

	// Server-side: issue signed token request
	server := qpub.NewRest(qpub.WithAPIKey(key))
	defer server.Reset()

	req, err := server.Auth.CreateTokenRequest(context.Background(), option.TokenOptions{
		Alias: "user-123",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Client-side: exchange token request for access token
	client := qpub.NewRest()
	defer client.Reset()

	resp, err := client.Auth.RequestToken(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("token length:", len(resp.Token))
}
