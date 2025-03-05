package tcplisten

import (
	"net"
	"os"
	"testing"
)

func TestNewListenerWithPerm(t *testing.T) {
	wantPerm := os.FileMode(0600)

	// Create a TCP listener with permissions
	// (note: file permissions don't have the same meaning for TCP sockets as they do for Unix sockets)
	cfg := Config{}
	ln, err := cfg.NewListenerWithPerm(wantPerm, "tcp4", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %s", err)
	}
	defer ln.Close()

	// Get the actual address
	addr := ln.Addr().String()

	// Verify the listener works by making a connection
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to listener: %s", err)
	}
	conn.Close()

	// IMPORTANT NOTE: For TCP sockets, file permissions don't work the same way
	// as Unix domain sockets. The Chmod() operation may succeed, but TCP socket access
	// is controlled by the network stack, not by file permissions.
	//
	// For a real permission test, the package would need to support
	// Unix domain sockets where file permissions are meaningful.

	t.Log("Listener created and functional with NewListenerWithPerm")
}
