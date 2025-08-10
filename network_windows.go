//go:build windows

package tcplisten

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
)

// CreateListener crates socket listener based on DSN definition.
func CreateListener(address string) (net.Listener, error) {
	dsn := strings.Split(address, "://")

	switch len(dsn) {
	case 1:
		// assume, that there is no prefix here [127.0.0.1:8000]
		return createTCPListener(dsn[0])
	case 2:
		// we got two part here, first part is the transport, second - address
		// [tcp://127.0.0.1:8000] OR [unix:///path/to/unix.socket] OR [error://path]
		// where error is wrong transport name
		switch dsn[0] {
		case "unix":
			// check of file exist. If exist, unlink
			err := unlinkExisting(dsn[1])
			if err != nil {
				return nil, err
			}

			return net.Listen(dsn[0], dsn[1])
		case "tcp":
			return createTCPListener(dsn[1])
			// not an tcp or unix
		default:
			return nil, fmt.Errorf("invalid Protocol ([tcp://]:6001, unix://file.sock), address: %s", address)
		}
		// wrong number of split parts
	default:
		return nil, fmt.Errorf("wrong number of parsed protocol parts, address: %s", address)
	}
}

func createTCPListener(addr string) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// checks if the socket file exists and unlinks if it isn't a directory
// this prevents errors from trying to use an already existing socket
func unlinkExisting(filename string) error {
	info, err := os.Stat(filename)

	// Skip unlinking if it doesn't exist or is a directory
	if errors.Is(err, os.ErrNotExist) || (info != nil && info.IsDir()) {
		return nil
	}

	if err != nil {
		return err
	}

	return syscall.Unlink(filename)
}
