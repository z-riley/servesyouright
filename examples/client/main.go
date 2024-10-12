package main

import (
	"fmt"
	"log"

	"github.com/z-riley/turdserve"
)

func main() {
	client := turdserve.NewClient()
	defer client.Destroy()

	errCh := make(chan error)
	defer close(errCh)

	client.Connect("127.0.0.1", 8080, errCh)
	go func() {
		for err := range errCh {
			if err != nil {
				log.Fatal("Client failed: ", err)
			}
		}
	}()

	client.SetCallback(func(b []byte) {
		fmt.Println(string(b))
	})

	client.Write([]byte("hello from client"))

	// Do other stuff...
	select {}
}
