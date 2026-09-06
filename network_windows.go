//go:build windows

package tcplisten

import (
	"context"
	"net"
)

func createTCPListener(addr string) (net.Listener, error) {
	var lc net.ListenConfig
	return lc.Listen(context.Background(), "tcp", addr)
}
