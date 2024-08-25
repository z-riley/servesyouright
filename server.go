package turdserve

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Server can communicate with a number of turdserve clients.
type Server struct {
	pool       sync.Map
	maxClients int
	callback   func(id int, msg []byte)
}

// NewServer constructs a new server with a set maximum number of concurrent clients.
// Call Run to start.
func NewServer(maxClients int) *Server {
	return &Server{
		pool:       sync.Map{},
		maxClients: maxClients,
		callback:   func(int, []byte) {},
	}
}

// Destroy gracefully shuts down the server.
func (s *Server) Destroy() {
	s.pool.Range(func(key any, value any) bool {
		err := value.(net.Conn).Close()
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to close connection ID %d", key.(int))
		}
		return true
	})
}

// SetCallback sets a callback which is executed when the server receives
// data from any client.
func (s *Server) SetCallback(cb func(id int, msg []byte)) *Server {
	s.callback = cb
	return s
}

// Run runs the server until an error occurs or Destroy is called.
func (s *Server) Run(host string, port uint16) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		// Check there's enough capacity in connection pool
		numConns := 0
		s.pool.Range(func(key, value any) bool {
			numConns++
			return true
		})
		if numConns >= s.maxClients {
			log.Warn().Msg("Max clients reached. Ignoring new client connection")
			conn.Write(append([]byte("Maximum number of clients reached"), delimChar))
			conn.Close()
			continue
		}

		// Save new connection to pool
		id := s.generateConnID()
		s.pool.Store(id, conn)
		log.Info().Msgf("Assigning incoming connection ID %d", id)

		go s.listenForever(conn, id)
	}
}

// ClientIDs returns a slice containing the ID of each client connection.
func (s *Server) GetClientIDs() []int {
	IDs := []int{}
	s.pool.Range(func(key, value any) bool {
		IDs = append(IDs, key.(int))
		return true
	})
	return IDs
}

// WriteToClient sends data to the client with the specified ID.
func (s *Server) WriteToClient(id int, b []byte) error {
	conn, ok := s.pool.Load(id)
	if !ok {
		return fmt.Errorf("invalid connection ID: %d", id)
	}

	_, err := conn.(net.Conn).Write(append(b, delimChar))
	return err
}

// generateConnID returns the lowest available connection ID in the pool.
func (s *Server) generateConnID() int {
	i := 0
	for {
		if _, taken := s.pool.Load(i); !taken {
			return i
		}
		i++
	}
}

const (
	delimChar         = '\n'
	heartbeatMsg      = "H34RTB34T" + string(delimChar)
	heartbeatInterval = 500 * time.Millisecond
)

// listenForever listens to a connection until it is closed.
func (s *Server) listenForever(conn net.Conn, id int) {
	// Exit the function if heartbeat not received
	timer := time.NewTimer(2 * heartbeatInterval)
	done := make(chan struct{})
	go func() {
		<-timer.C
		log.Info().Msgf("Closing connection %d as heartbeat not recevied", id)
		log.Info().Msgf("Removing connection %d from pool", id)
		s.pool.Delete(id)
		conn.Close()
		done <- struct{}{}
	}()

	reader := bufio.NewReader(conn)
	for {
		select {
		case <-done:
			return
		default:
			message, err := reader.ReadBytes('\n')
			if err != nil {
				s.closeConnection(id)
				return
			}
			if string(message) == heartbeatMsg {
				timer.Reset(2 * heartbeatInterval)
				log.Debug().Msgf("Received heartbeat from connection id: %d", id)
			} else {
				s.callback(id, message)
			}
		}
	}
}

// closeConnection closes the connection in the pool with the given ID.
func (s *Server) closeConnection(id int) {
	conn, ok := s.pool.Load(id)
	if !ok {
		log.Warn().Msg("Could not close connection with ID %d as it does not exist")
		return
	}
	conn.(net.Conn).Close()
}
