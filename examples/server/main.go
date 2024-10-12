package main

import (
	"fmt"
	"log"

	"github.com/z-riley/turdserve"
)

func main() {
	maxClients := 2
	server := turdserve.NewServer(maxClients)
	defer server.Destroy()

	server.SetCallback(func(id int, msg []byte) {
		fmt.Printf("Server received message from connection %d: %s", id, string(msg))
	})

	// Listen for errors
	errCh := make(chan error)
	go func() {
		for err := range errCh {
			if err != nil {
				log.Fatal("Server error: ", err)
			}
		}
	}()

	// Start the server
	if err := server.Start("0.0.0.0", 8080, errCh); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Do other stuff...
	select {}
}
