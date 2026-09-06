//go:build linux || darwin || freebsd || windows

package tcplisten

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// UnixSocketOptions sets attributes on a filesystem UNIX socket.
// Omitted attributes keep their operating system defaults.
type UnixSocketOptions struct {
	// Mode is an octal string from "0000" through "0777". Empty means no change.
	Mode string `mapstructure:"mode"`
	// UID sets the socket owner. Nil means no change. Zero is a valid ID.
	UID *int `mapstructure:"uid"`
	// GID sets the socket group. Nil means no change. Zero is a valid ID.
	GID *int `mapstructure:"gid"`
}

// Validate checks the options and address without filesystem access.
// A nil receiver is valid for every address.
// Non-nil options require a filesystem UNIX socket on Linux, macOS, or FreeBSD.
func (o *UnixSocketOptions) Validate(address string) error {
	_, err := o.validate(address)
	return err
}

func (o *UnixSocketOptions) validate(address string) (os.FileMode, error) {
	if o == nil {
		return 0, nil
	}
	if runtime.GOOS == "windows" {
		return 0, errors.New("unix socket attributes are not supported on Windows")
	}
	network, path, prefixed := strings.Cut(address, "://")
	if !prefixed || network != "unix" || path == "" || strings.Contains(path, "://") || strings.ContainsRune(path, 0) || isAbstractSocket(path) {
		return 0, fmt.Errorf("unix socket options require a filesystem unix:// address: %q", address)
	}

	var mode os.FileMode
	if o.Mode != "" {
		value, err := strconv.ParseUint(o.Mode, 8, 9)
		if err != nil || len(o.Mode) != 4 || o.Mode[0] != '0' {
			return 0, fmt.Errorf("invalid unix socket mode %q: use an octal string from 0000 through 0777", o.Mode)
		}
		mode = os.FileMode(value)
	}
	if o.UID != nil && (*o.UID < 0 || uint64(*o.UID) >= (1<<32)-1) {
		return 0, fmt.Errorf("invalid unix socket uid %d: must be between 0 and 4294967294", *o.UID)
	}
	if o.GID != nil && (*o.GID < 0 || uint64(*o.GID) >= (1<<32)-1) {
		return 0, fmt.Errorf("invalid unix socket gid %d: must be between 0 and 4294967294", *o.GID)
	}
	return mode, nil
}

func (o *UnixSocketOptions) apply(path string, mode os.FileMode) error {
	if o == nil {
		return nil
	}
	if o.UID != nil || o.GID != nil {
		uid, gid := -1, -1
		if o.UID != nil {
			uid = *o.UID
		}
		if o.GID != nil {
			gid = *o.GID
		}
		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("chown unix socket %q: %w", path, err)
		}
	}
	if o.Mode != "" {
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("chmod unix socket %q: %w", path, err)
		}
	}
	return nil
}

func isAbstractSocket(path string) bool {
	return (runtime.GOOS == "linux" || runtime.GOOS == "windows") && len(path) > 0 && (path[0] == '@' || path[0] == 0)
}
