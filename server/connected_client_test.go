package server

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConnectedClient_String(t *testing.T) {
	client := ConnectedClient{
		RemoteAddr:     "192.168.1.10:54321",
		ConnectedAt:    time.Now().Add(-2 * time.Hour),
		RxTransactions: 1523,
		TxTransactions: 1520,
	}

	s := client.String()

	// Verify all expected components are present
	if !strings.Contains(s, "192.168.1.10:54321") {
		t.Errorf("String() missing remote address, got: %s", s)
	}
	if !strings.Contains(s, "connected") {
		t.Errorf("String() missing 'connected' label, got: %s", s)
	}
	if !strings.Contains(s, "rx: 1523") {
		t.Errorf("String() missing rx count, got: %s", s)
	}
	if !strings.Contains(s, "tx: 1520") {
		t.Errorf("String() missing tx count, got: %s", s)
	}
}

func TestConnectedClient_String_ZeroCounts(t *testing.T) {
	client := ConnectedClient{
		RemoteAddr:  "10.0.0.1:12345",
		ConnectedAt: time.Now(),
	}

	s := client.String()

	if !strings.Contains(s, "rx: 0") {
		t.Errorf("String() should show rx: 0 for new client, got: %s", s)
	}
	if !strings.Contains(s, "tx: 0") {
		t.Errorf("String() should show tx: 0 for new client, got: %s", s)
	}
}

func TestClientConn_AtomicCounters(t *testing.T) {
	client := &clientConn{
		remoteAddr:  "127.0.0.1:9999",
		connectedAt: time.Now(),
	}

	client.rxCount.Add(1)
	client.rxCount.Add(1)
	client.rxCount.Add(1)
	client.txCount.Add(1)
	client.txCount.Add(1)

	if client.rxCount.Load() != 3 {
		t.Errorf("Expected rxCount=3, got %d", client.rxCount.Load())
	}
	if client.txCount.Load() != 2 {
		t.Errorf("Expected txCount=2, got %d", client.txCount.Load())
	}
}

func TestTCPServer_ConnectedClients_Empty(t *testing.T) {
	srv := NewTCPServer("127.0.0.1", WithServerPort(0))

	clients := srv.ConnectedClients()
	if len(clients) != 0 {
		t.Errorf("Expected 0 connected clients, got %d", len(clients))
	}
}

func TestTCPServer_ConnectedClients_Snapshot(t *testing.T) {
	srv := NewTCPServer("127.0.0.1", WithServerPort(0))

	// Manually inject a tracked client to test snapshot logic
	// without needing a real TCP connection
	now := time.Now()
	client := &clientConn{
		remoteAddr:  "10.0.0.5:40000",
		connectedAt: now,
	}
	client.rxCount.Store(100)
	client.txCount.Store(99)

	srv.clientsMutex.Lock()
	srv.clients["10.0.0.5:40000"] = client
	srv.clientsMutex.Unlock()

	snapshots := srv.ConnectedClients()
	if len(snapshots) != 1 {
		t.Fatalf("Expected 1 connected client, got %d", len(snapshots))
	}

	snap := snapshots[0]
	if snap.RemoteAddr != "10.0.0.5:40000" {
		t.Errorf("Expected RemoteAddr=10.0.0.5:40000, got %s", snap.RemoteAddr)
	}
	if snap.ConnectedAt != now {
		t.Errorf("Expected ConnectedAt=%v, got %v", now, snap.ConnectedAt)
	}
	if snap.RxTransactions != 100 {
		t.Errorf("Expected RxTransactions=100, got %d", snap.RxTransactions)
	}
	if snap.TxTransactions != 99 {
		t.Errorf("Expected TxTransactions=99, got %d", snap.TxTransactions)
	}
}

func TestWithOnClientConnect(t *testing.T) {
	var mu sync.Mutex
	var got ConnectedClient
	called := false

	srv := NewTCPServer("127.0.0.1",
		WithServerPort(0),
		WithOnClientConnect(func(c ConnectedClient) {
			mu.Lock()
			defer mu.Unlock()
			got = c
			called = true
		}),
	)

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	// Connect a real TCP client
	addr := srv.listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Give the accept loop time to process
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !called {
		t.Fatal("OnClientConnect callback was not called")
	}
	if got.RemoteAddr == "" {
		t.Error("OnClientConnect received empty RemoteAddr")
	}
	if got.ConnectedAt.IsZero() {
		t.Error("OnClientConnect received zero ConnectedAt")
	}
	if got.RxTransactions != 0 || got.TxTransactions != 0 {
		t.Errorf("OnClientConnect should have zero transactions, got rx=%d tx=%d",
			got.RxTransactions, got.TxTransactions)
	}
}

func TestWithOnClientDisconnect(t *testing.T) {
	var mu sync.Mutex
	var got ConnectedClient
	called := make(chan struct{})

	srv := NewTCPServer("127.0.0.1",
		WithServerPort(0),
		WithOnClientDisconnect(func(c ConnectedClient) {
			mu.Lock()
			defer mu.Unlock()
			got = c
			close(called)
		}),
	)

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	// Connect and immediately close
	addr := srv.listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	// Let the server register the connection
	time.Sleep(50 * time.Millisecond)
	conn.Close()

	// Wait for disconnect callback
	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("OnClientDisconnect callback was not called within timeout")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.RemoteAddr == "" {
		t.Error("OnClientDisconnect received empty RemoteAddr")
	}
	if got.ConnectedAt.IsZero() {
		t.Error("OnClientDisconnect received zero ConnectedAt")
	}
}

func TestNilCallbacksDoNotPanic(t *testing.T) {
	// Server with no callbacks set should not panic
	srv := NewTCPServer("127.0.0.1", WithServerPort(0))

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	addr := srv.listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	conn.Close()

	// Give time for connect+disconnect to process without panicking
	time.Sleep(100 * time.Millisecond)
}
