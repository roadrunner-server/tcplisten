//go:build linux || darwin || freebsd

package tcplisten

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnixSocketApplyError(t *testing.T) {
	uid := os.Getuid()
	for _, tt := range []struct {
		name    string
		options UnixSocketOptions
	}{
		{name: "chown", options: UnixSocketOptions{UID: &uid}},
		{name: "chmod", options: UnixSocketOptions{Mode: "0600"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing.sock")
			err := tt.options.apply(path, 0o600)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected a wrapped filesystem error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.name) || !strings.Contains(err.Error(), path) {
				t.Fatalf("missing operation or path in error: %v", err)
			}
		})
	}
}
