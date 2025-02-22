package servesyouright

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

var LoggingEnabled bool = false

// Server can communicate with a number of servesyouright clients.
type Server struct {
	pool               sync.Map                 // holds client connections
	maxClients         int                      // the maximum number of clients
	callback           func(id int, msg []byte) // executes on receipt of data
	connectCallback    func(id int)             // executes on client connection
	disconnectCallback func(id int)             // executes on client disconnection
	listener           net.Listener             // TCP listener
}

// NewServer constructs a new server with a set maximum number of concurrent clients.
// Call Run to start.
func NewServer(maxClients int) *Server {
	return &Server{
		pool:               sync.Map{},
		maxClients:         maxClients,
		callback:           func(int, []byte) {},
		connectCallback:    func(int) {},
		disconnectCallback: func(int) {},
		listener:           nil,
	}
}

// Destroy gracefully shuts down the server.
func (s *Server) Destroy() {
	s.pool.Range(func(key any, value any) bool {
		err := value.(net.Conn).Close()
		if err != nil {
			logf("Failed to close connection ID %d", key.(int))
		}
		return true
	})
	s.listener.Close()
}

// SetCallback sets a callback which is executed when the server receives
// data from any client.
func (s *Server) SetCallback(cb func(id int, msg []byte)) *Server {
	s.callback = cb
	return s
}

// SetConnectCallback sets a callback which is executed when the a client connects
// to the server.
func (s *Server) SetConnectCallback(cb func(id int)) *Server {
	s.connectCallback = cb
	return s
}

// SetDisconnectCallback sets a callback which is executed when the a client disconnects
// from the server.
func (s *Server) SetDisconnectCallback(cb func(id int)) *Server {
	s.disconnectCallback = cb
	return s
}

// Start runs the server forever. An error is returned if the initial listen fails.
// Susequent errors are sent down the error channel.
//
// The error channel should be opened before Start is called.
func (s *Server) Start(host string, port uint16, errCh chan error) error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}

	// Process incoming connections
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				errCh <- fmt.Errorf("failed to accept connection: %w", err)
				continue
			}

			// Check there's enough capacity in connection pool
			numConns := 0
			s.pool.Range(
				func(key, value any) bool {
					numConns++
					return true
				})

			// Reject connection if pool is full
			if numConns >= s.maxClients {
				logf("Max clients reached. Ignoring new client connection")
				conn.Write(append([]byte("Maximum number of clients reached"), delimChar))
				conn.Close()
				continue
			}

			// Save new connection to pool
			id := s.generateConnID()
			s.pool.Store(id, conn)
			logf("Assigning incoming connection ID %d", id)

			s.connectCallback(id)

			go s.listenForever(conn, id)
		}
	}()

	return nil
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
		logf("Closing connection %d as heartbeat not recevied", id)
		s.closeConnection(id)
		logf("Removing connection %d from pool", id)
		s.pool.Delete(id)
		s.disconnectCallback(id)
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
				logf("Received heartbeat from connection id: %d", id)
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
		logf("Could not close connection with ID %d as it does not exist", id)
		return
	}
	conn.(net.Conn).Close()
}

// logf logs a messsage if loggingEnabled is set.
func logf(format string, v ...any) {
	if LoggingEnabled {
		log.Printf(format, v...)
	}
}
