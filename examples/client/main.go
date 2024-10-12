package main

import (
	"fmt"
	"log"

	"github.com/z-riley/turdserve"
)

func main() {
	client := turdserve.NewClient()
	defer client.Destroy()

	// Listen for errors
	errCh := make(chan error)
	go func() {
		for err := range errCh {
			if err != nil {
				log.Fatal("Client failed: ", err)
			}
		}
	}()

	// Connect to the server
	if err := client.Connect("127.0.0.1", 8080, errCh); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	client.SetCallback(func(b []byte) {
		fmt.Println("Received from server:", string(b))
	})

	if err := client.Write([]byte("hello from client")); err != nil {
		log.Fatalf("Write to server failed: %v", err)
	}

	// Do other stuff...
	select {}
}
