package main

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/z-riley/turdserve"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	maxClients := 2
	server := turdserve.NewServer(maxClients)
	defer server.Destroy()

	server.SetCallback(func(id int, msg []byte) {
		fmt.Printf("Server received message from connection %d: %s", id, string(msg))
	})

	if err := server.Run("0.0.0.0", 8080); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
