//go:build windows

package tcplisten

import (
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/roadrunner-server/tcplisten"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixSocketWindowsFilesystem(t *testing.T) {
	t.Chdir(t.TempDir())
	probe, err := net.Listen("unix", "probe")
	if err != nil {
		t.Skipf("Native AF_UNIX is unavailable: %v", err)
	}
	t.Cleanup(func() { _ = probe.Close() })
	require.NoError(t, probe.Close())

	for _, factory := range []struct {
		name   string
		create func(string) (net.Listener, error)
	}{
		{"legacy", tcplisten.CreateListener},
		{"nil_options", func(address string) (net.Listener, error) {
			return tcplisten.CreateListenerWithOptions(address, nil)
		}},
	} {
		t.Run(factory.name, func(t *testing.T) {
			for _, path := range []string{"socket", "./@socket"} {
				t.Run(path, func(t *testing.T) {
					original, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
					require.NoError(t, err)
					original.SetUnlinkOnClose(false)
					t.Cleanup(func() { _ = original.Close() })
					require.NoError(t, original.Close())
					info, err := os.Lstat(path)
					require.NoError(t, err)
					assert.Equal(t, os.ModeSocket, info.Mode().Type())

					for _, state := range []string{"replace", "create"} {
						t.Run(state, func(t *testing.T) {
							listener, err := factory.create("unix://" + path)
							require.NoError(t, err)
							t.Cleanup(func() { _ = listener.Close() })
							assert.Equal(t, original.Addr(), listener.Addr())
							info, err := os.Lstat(path)
							require.NoError(t, err)
							assert.Equal(t, os.ModeSocket, info.Mode().Type())
							conn, err := net.DialTimeout("unix", path, time.Second)
							require.NoError(t, err)
							require.NoError(t, conn.Close())
							require.NoError(t, listener.Close())
							_, err = os.Lstat(path)
							assert.ErrorIs(t, err, os.ErrNotExist)
						})
					}
				})
			}
		})
	}
}

func TestUnixSocketWindowsEncodedAddresses(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, tc := range []struct {
		name   string
		prefix string
	}{{"at", "@"}, {"nul", "\x00"}} {
		t.Run(tc.name, func(t *testing.T) {
			name := fmt.Sprintf("tcplisten-%d-%s", os.Getpid(), tc.name)
			markers := make(map[string]os.FileInfo, 2)
			for _, path := range []string{name, "@" + name} {
				require.NoError(t, os.WriteFile(path, []byte("keep"), 0600))
				info, err := os.Lstat(path)
				require.NoError(t, err)
				markers[path] = info
			}
			address := tc.prefix + name
			baseline, baselineErr := net.Listen("unix", address)
			var wantAddr net.Addr
			if baselineErr == nil {
				t.Cleanup(func() { _ = baseline.Close() })
				wantAddr = baseline.Addr()
				require.NoError(t, baseline.Close())
			} else {
				for errors.Unwrap(baselineErr) != nil {
					baselineErr = errors.Unwrap(baselineErr)
				}
			}

			for _, factory := range []struct {
				name   string
				create func(string) (net.Listener, error)
			}{
				{"legacy", tcplisten.CreateListener},
				{"nil_options", func(address string) (net.Listener, error) {
					return tcplisten.CreateListenerWithOptions(address, nil)
				}},
			} {
				t.Run(factory.name, func(t *testing.T) {
					listener, err := factory.create("unix://" + address)
					if listener != nil {
						t.Cleanup(func() { _ = listener.Close() })
					}
					if baselineErr != nil {
						assert.ErrorIs(t, err, baselineErr)
						assert.Nil(t, listener)
					} else {
						require.NoError(t, err)
						assert.Equal(t, wantAddr, listener.Addr())
						require.NoError(t, listener.Close())
					}
					for path, before := range markers {
						after, err := os.Lstat(path)
						require.NoError(t, err)
						assert.True(t, os.SameFile(before, after))
						assert.Equal(t, before.Mode(), after.Mode())
						contents, err := os.ReadFile(path)
						require.NoError(t, err)
						assert.Equal(t, "keep", string(contents))
					}
				})
			}
		})
	}
}
