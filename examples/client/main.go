package main

import (
	"fmt"
	"log"

	"github.com/z-riley/turdserve"
)

func main() {
	client := turdserve.NewClient()

	if err := client.Connect("127.0.0.1", 8080); err != nil {
		log.Fatal("Client failed to connect", err)
	}

	client.SetCallback(func(b []byte) {
		fmt.Println(string(b))
	})

	client.Write([]byte("hello from client"))

	// Do other stuff...
	select {}
}
