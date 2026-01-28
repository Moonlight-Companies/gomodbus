package server

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

// clientConn is the internal per-connection tracking state.
// It contains atomics and a net.Conn, so it must not be copied.
type clientConn struct {
	remoteAddr  string
	connectedAt time.Time
	conn        net.Conn
	rxCount     atomic.Uint64
	txCount     atomic.Uint64
}

// ConnectedClient is a snapshot of a connected client's state.
// Returned by TCPServer.ConnectedClients(). Safe to copy and store.
type ConnectedClient struct {
	// RemoteAddr is the remote address of the connected client.
	RemoteAddr string

	// ConnectedAt is the time the client connected.
	ConnectedAt time.Time

	// RxTransactions is the number of requests received from this client.
	RxTransactions uint64

	// TxTransactions is the number of responses sent to this client.
	TxTransactions uint64
}

// String returns a human-readable summary of the connected client.
func (c ConnectedClient) String() string {
	duration := time.Since(c.ConnectedAt).Truncate(time.Second)
	return fmt.Sprintf("%s | connected %s | rx: %d tx: %d", c.RemoteAddr, duration, c.RxTransactions, c.TxTransactions)
}
