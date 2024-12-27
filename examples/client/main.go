package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/z-riley/servesyouright"
)

func main() {
	client := servesyouright.NewClient()
	defer client.Destroy()

	// Create context and send cancellation signal after 3s
	ctx, cancelFunc := context.WithCancel(context.Background())
	timer := time.NewTimer(3 * time.Second)

	// Listen for errors
	errCh := make(chan error)
	go func() {
		for err := range errCh {
			if err != nil {
				log.Fatal("Client error: ", err)
			}
		}
	}()

	// Connect to the server
	if err := client.Connect(ctx, "127.0.0.1", 8080, errCh); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	client.SetCallback(func(b []byte) {
		fmt.Println("Received from server:", string(b))
	})

	if err := client.Write([]byte("hello from client")); err != nil {
		log.Fatalf("Write to server failed: %v", err)
	}

	// Cancel the client connection after the timer expires
	<-timer.C
	cancelFunc()
}
