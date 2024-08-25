package main

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/z-riley/turdserve"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	client := turdserve.NewClient()
	if err := client.Connect("0.0.0.0", 8080); err != nil {
		log.Fatal().Err(err).Msg("Client failed to connect")
	}

	client.SetCallback(func(b []byte) {
		fmt.Println(string(b))
	})

	client.Write([]byte("hello from client"))

	// Do other stuff...
	select {}
}
