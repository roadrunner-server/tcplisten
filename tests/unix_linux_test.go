//go:build linux

package tcplisten

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/roadrunner-server/tcplisten"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixSocketAbstract(t *testing.T) {
	dir := socketTempDir(t)
	t.Chdir(dir)
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
			for _, tc := range []struct {
				name   string
				prefix string
			}{{"at", "@"}, {"nul", "\x00"}} {
				t.Run(tc.name, func(t *testing.T) {
					name := filepath.Base(dir) + "-" + factory.name + "-" + tc.name
					file := "@" + name
					require.NoError(t, os.WriteFile(file, []byte("keep"), 0600))
					before, err := os.Lstat(file)
					require.NoError(t, err)
					address := "unix://" + tc.prefix + name
					listener, err := factory.create(address)
					require.NoError(t, err)
					t.Cleanup(func() { _ = listener.Close() })
					conn, err := net.DialTimeout("unix", tc.prefix+name, time.Second)
					require.NoError(t, err)
					require.NoError(t, conn.Close())
					for _, options := range []*tcplisten.UnixSocketOptions{{}, {Mode: "0600"}} {
						rejected, err := tcplisten.CreateListenerWithOptions(address, options)
						if rejected != nil {
							t.Cleanup(func() { _ = rejected.Close() })
						}
						assert.Error(t, err)
						assert.Nil(t, rejected)
					}
					require.NoError(t, listener.Close())
					after, err := os.Lstat(file)
					require.NoError(t, err)
					assert.True(t, os.SameFile(before, after))
					assert.Equal(t, before.Mode(), after.Mode())
					contents, err := os.ReadFile(file)
					require.NoError(t, err)
					assert.Equal(t, "keep", string(contents))
					replacement, err := net.Listen("unix", tc.prefix+name)
					require.NoError(t, err)
					require.NoError(t, replacement.Close())
				})
			}
		})
	}
}

func TestUnixSocketAutobind(t *testing.T) {
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
			listener, err := factory.create("unix://")
			require.NoError(t, err)
			t.Cleanup(func() { _ = listener.Close() })
			require.Greater(t, len(listener.Addr().String()), 1)
			require.Equal(t, byte('@'), listener.Addr().String()[0])
			conn, err := net.DialTimeout("unix", listener.Addr().String(), time.Second)
			require.NoError(t, err)
			require.NoError(t, conn.Close())
		})
	}
}

func TestUnixSocketCrossUserAccess(t *testing.T) {
	const env = "TCPLISTEN_TEST_CROSS_USER"
	if access := os.Getenv(env); access != "" {
		conn, err := net.DialTimeout("unix", "socket", time.Second)
		if conn != nil {
			require.NoError(t, conn.Close())
		}
		if access == "deny" {
			require.ErrorIs(t, err, syscall.EACCES)
		} else {
			require.Equal(t, "allow", access)
			require.NoError(t, err)
		}
		return
	}
	if os.Geteuid() != 0 {
		t.Skip("Root is required to set socket ownership and child credentials")
	}
	dir := socketTempDir(t)
	require.NoError(t, os.Chmod(dir, 0755))

	// Go build directories can deny search permission to child users.
	executable, err := os.Executable()
	require.NoError(t, err)
	contents, err := os.ReadFile(executable)
	require.NoError(t, err)
	childBinary := filepath.Join(dir, "access.test")
	require.NoError(t, os.WriteFile(childBinary, contents, 0600)) //nolint:gosec // G703: Copy this test binary to a fixed name in the root-owned temporary directory.
	require.NoError(t, os.Chmod(childBinary, 0755))

	uid, gid := 65534, 65533
	path := filepath.Join(dir, "socket")
	listener, err := tcplisten.CreateListenerWithOptions("unix://"+path, &tcplisten.UnixSocketOptions{Mode: "0660", UID: &uid, GID: &gid})
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	info, err := os.Lstat(path)
	require.NoError(t, err)
	require.Equal(t, os.ModeSocket|0660, info.Mode())
	stat := info.Sys().(*syscall.Stat_t)
	require.Equal(t, int64(uid), int64(stat.Uid))
	require.Equal(t, int64(gid), int64(stat.Gid))

	for _, tc := range []struct {
		name   string
		uid    uint32
		gid    uint32
		access string
	}{
		{"owner", 65534, 65532, "allow"},
		{"group", 65533, 65533, "allow"},
		{"unrelated", 65532, 65532, "deny"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, childBinary, "-test.run=^TestUnixSocketCrossUserAccess$", "-test.count=1", "-test.timeout=10s")
			command.Dir = dir
			command.Env = append(os.Environ(), env+"="+tc.access)
			command.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{Uid: tc.uid, Gid: tc.gid, Groups: []uint32{}},
			}
			output, err := command.CombinedOutput()
			require.NoError(t, err, "UID %d GID %d:\n%s", tc.uid, tc.gid, output)
		})
	}
}
