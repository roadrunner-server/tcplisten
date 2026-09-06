//go:build linux || darwin || freebsd || windows

package tcplisten

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
)

// CreateListener creates a TCP or UNIX listener from an address.
// A TCP address can omit the tcp:// prefix. UNIX addresses require unix://.
// Filesystem socket permissions and ownership use the operating system defaults.
func CreateListener(address string) (net.Listener, error) {
	return CreateListenerWithOptions(address, nil)
}

// CreateListenerWithOptions creates a listener with optional UNIX socket attributes.
// It validates the options before it changes the filesystem.
// It closes the listener if an attribute change fails.
// Attributes are set after listen, so clients can connect before setup completes.
// Parent directories must restrict initial access and prevent path replacement.
func CreateListenerWithOptions(address string, options *UnixSocketOptions) (net.Listener, error) {
	mode, err := options.validate(address)
	if err != nil {
		return nil, err
	}

	network, path, prefixed := strings.Cut(address, "://")
	if !prefixed {
		return createTCPListener(address)
	}
	if strings.Contains(path, "://") {
		return nil, fmt.Errorf("wrong number of parsed protocol parts, address: %s", address)
	}
	switch network {
	case "tcp":
		return createTCPListener(path)
	case "unix":
		return createUnixListener(path, options, mode)
	default:
		return nil, fmt.Errorf("invalid protocol ([tcp://]:6001, unix://file.sock), address: %s", address)
	}
}

func createUnixListener(path string, options *UnixSocketOptions, mode os.FileMode) (net.Listener, error) {
	if path != "" && !isAbstractSocket(path) {
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect unix socket %q: %w", path, err)
		}
		if err == nil {
			if info.Mode()&os.ModeSocket == 0 {
				return nil, fmt.Errorf("refuse to replace non-socket path %q", path)
			}
			if err = os.Remove(path); err != nil {
				return nil, fmt.Errorf("remove unix socket %q: %w", path, err)
			}
		}
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "unix", path)
	if err != nil {
		return nil, err
	}
	if err = options.apply(path, mode); err != nil {
		// Go does not unlink BSD filesystem names that start with @.
		if strings.HasPrefix(path, "@") && !isAbstractSocket(path) {
			_ = os.Remove(path)
		}
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}
