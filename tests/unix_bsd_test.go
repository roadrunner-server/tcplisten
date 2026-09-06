//go:build darwin || freebsd

package tcplisten

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/roadrunner-server/tcplisten"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixSocketLeadingAt(t *testing.T) {
	for _, factory := range []struct {
		name   string
		create func(string) (net.Listener, error)
	}{
		{"legacy", tcplisten.CreateListener},
		{"nil_options", func(address string) (net.Listener, error) {
			return tcplisten.CreateListenerWithOptions(address, nil)
		}},
		{"options", func(address string) (net.Listener, error) {
			return tcplisten.CreateListenerWithOptions(address, &tcplisten.UnixSocketOptions{Mode: "0600"})
		}},
	} {
		t.Run(factory.name, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				path   string
				unlink bool
			}{
				{"raw", "@socket", false},
				{"length_102", "@" + strings.Repeat("s", 101), false},
				{"length_103", "@" + strings.Repeat("s", 102), false},
				{"explicit_relative", "./@socket", true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Chdir(socketTempDir(t))
					original, err := net.ListenUnix("unix", &net.UnixAddr{Name: tc.path, Net: "unix"})
					require.NoError(t, err)
					original.SetUnlinkOnClose(false)
					t.Cleanup(func() { _ = original.Close() })
					before, err := os.Lstat(tc.path)
					require.NoError(t, err)

					listener, err := factory.create("unix://" + tc.path)
					require.NoError(t, err)
					t.Cleanup(func() { _ = listener.Close() })
					assert.Equal(t, tc.path, listener.Addr().String())
					info, err := os.Lstat(tc.path)
					require.NoError(t, err)
					assert.Equal(t, os.ModeSocket, info.Mode().Type())
					assert.False(t, os.SameFile(before, info))
					if factory.name == "options" {
						assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
					}

					rejected, err := tcplisten.CreateListenerWithOptions("unix://"+tc.path, &tcplisten.UnixSocketOptions{Mode: "600"})
					if rejected != nil {
						t.Cleanup(func() { _ = rejected.Close() })
					}
					assert.Error(t, err)
					assert.Nil(t, rejected)
					after, err := os.Lstat(tc.path)
					require.NoError(t, err)
					assert.True(t, os.SameFile(info, after))
					assert.Equal(t, info.Mode(), after.Mode())
					conn, err := net.DialTimeout("unix", tc.path, time.Second)
					require.NoError(t, err)
					require.NoError(t, conn.Close())
					require.NoError(t, listener.Close())
					after, err = os.Lstat(tc.path)
					if tc.unlink {
						assert.ErrorIs(t, err, os.ErrNotExist)
					} else {
						// Go leaves raw @ filesystem paths in place after a successful close.
						require.NoError(t, err)
						assert.True(t, os.SameFile(info, after))
					}
				})
			}
		})
	}
}
