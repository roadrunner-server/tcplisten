//go:build linux || darwin || freebsd

package tcplisten

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/roadrunner-server/tcplisten"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func socketTempDir(t *testing.T) string {
	t.Helper()
	// The macOS temporary directory can exceed the UNIX socket path limit.
	dir, err := os.MkdirTemp("/tmp", "tcplisten-")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(dir)) })
	return dir
}

func TestUnixSocketAttributes(t *testing.T) {
	dir := socketTempDir(t)
	require.NoError(t, os.Chmod(dir, 0750))
	parent, err := os.Stat(dir)
	require.NoError(t, err)
	baseline, err := net.Listen("unix", filepath.Join(dir, "baseline"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = baseline.Close() })
	info, err := os.Lstat(baseline.Addr().String())
	require.NoError(t, err)
	stat := info.Sys().(*syscall.Stat_t)
	uid, gid := os.Geteuid(), os.Getegid()

	for _, tc := range []struct {
		name    string
		options *tcplisten.UnixSocketOptions
		legacy  bool
	}{
		{name: "legacy", legacy: true},
		{name: "nil_options"},
		{name: "empty_options", options: &tcplisten.UnixSocketOptions{}},
		{name: "uid", options: &tcplisten.UnixSocketOptions{UID: &uid}},
		{name: "gid", options: &tcplisten.UnixSocketOptions{GID: &gid}},
		{name: "uid_gid", options: &tcplisten.UnixSocketOptions{UID: &uid, GID: &gid}},
		{name: "uid_gid_mode", options: &tcplisten.UnixSocketOptions{UID: &uid, GID: &gid, Mode: "0660"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			var listener net.Listener
			var err error
			if tc.legacy {
				listener, err = tcplisten.CreateListener("unix://" + path)
			} else {
				listener, err = tcplisten.CreateListenerWithOptions("unix://"+path, tc.options)
			}
			require.NoError(t, err)
			t.Cleanup(func() { _ = listener.Close() })
			actual, err := os.Lstat(path)
			require.NoError(t, err)
			wantMode, wantUID, wantGID := info.Mode(), int64(stat.Uid), int64(stat.Gid)
			if tc.options != nil {
				if tc.options.Mode != "" {
					wantMode = os.ModeSocket | 0660
				}
				if tc.options.UID != nil {
					wantUID = int64(uid)
				}
				if tc.options.GID != nil {
					wantGID = int64(gid)
				}
			}
			actualStat := actual.Sys().(*syscall.Stat_t)
			assert.Equal(t, wantMode, actual.Mode())
			assert.Equal(t, wantUID, int64(actualStat.Uid))
			assert.Equal(t, wantGID, int64(actualStat.Gid))
			require.NoError(t, listener.Close())
			_, err = os.Lstat(path)
			assert.ErrorIs(t, err, os.ErrNotExist)
		})
	}
	actualParent, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, parent.Mode(), actualParent.Mode())
	assert.Equal(t, parent.Sys().(*syscall.Stat_t).Uid, actualParent.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t, parent.Sys().(*syscall.Stat_t).Gid, actualParent.Sys().(*syscall.Stat_t).Gid)
}

func TestUnixSocketConcurrentModes(t *testing.T) {
	t.Parallel()
	dir := socketTempDir(t)
	type result struct {
		listener net.Listener
		err      error
		path     string
		mode     os.FileMode
	}
	modes := []os.FileMode{0000, 0600, 0660, 0777}
	results := make(chan result, len(modes))
	for _, mode := range modes {
		go func() {
			path := filepath.Join(dir, fmt.Sprintf("%04o", mode))
			listener, err := tcplisten.CreateListenerWithOptions("unix://"+path, &tcplisten.UnixSocketOptions{Mode: fmt.Sprintf("%04o", mode)})
			results <- result{listener, err, path, mode}
		}()
	}
	sockets := make([]result, 0, len(modes))
	for range modes {
		socket := <-results
		if socket.listener != nil {
			t.Cleanup(func() { _ = socket.listener.Close() })
		}
		sockets = append(sockets, socket)
	}
	for _, socket := range sockets {
		require.NoError(t, socket.err)
		info, err := os.Lstat(socket.path)
		require.NoError(t, err)
		assert.Equal(t, os.ModeSocket|socket.mode, info.Mode())
		require.NoError(t, socket.listener.Close())
		_, err = os.Lstat(socket.path)
		assert.ErrorIs(t, err, os.ErrNotExist)
	}
}

func TestUnixSocketInvalidOptionsPreserveSocket(t *testing.T) {
	path := filepath.Join(socketTempDir(t), "socket")
	original, err := net.Listen("unix", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = original.Close() })
	before, err := os.Lstat(path)
	require.NoError(t, err)
	negative := -1
	for _, tc := range []struct {
		name    string
		address string
		options *tcplisten.UnixSocketOptions
	}{
		{"mode", "unix://" + path, &tcplisten.UnixSocketOptions{Mode: "600"}},
		{"special_bits", "unix://" + path, &tcplisten.UnixSocketOptions{Mode: "4600"}},
		{"uid", "unix://" + path, &tcplisten.UnixSocketOptions{UID: &negative}},
		{"gid", "unix://" + path, &tcplisten.UnixSocketOptions{GID: &negative}},
		{"malformed_address", "unix://" + path + "://extra", &tcplisten.UnixSocketOptions{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := tcplisten.CreateListenerWithOptions(tc.address, tc.options)
			if listener != nil {
				t.Cleanup(func() { _ = listener.Close() })
			}
			assert.Error(t, err)
			assert.Nil(t, listener)
			after, err := os.Lstat(path)
			require.NoError(t, err)
			assert.True(t, os.SameFile(before, after))
			assert.Equal(t, before.Mode(), after.Mode())
			assert.Equal(t, before.Sys().(*syscall.Stat_t).Uid, after.Sys().(*syscall.Stat_t).Uid)
			assert.Equal(t, before.Sys().(*syscall.Stat_t).Gid, after.Sys().(*syscall.Stat_t).Gid)
		})
	}
}

func TestUnixSocketPreservesNonSockets(t *testing.T) {
	dir := socketTempDir(t)
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("keep"), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "directory"), 0750))
	socketPath := filepath.Join(dir, "target")
	socket, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = socket.Close() })
	socketInfo, err := os.Lstat(socketPath)
	require.NoError(t, err)
	for name, target := range map[string]string{
		"file_link": "file", "directory_link": "directory", "socket_link": "target", "dangling_link": "missing", "loop": "loop",
	} {
		require.NoError(t, os.Symlink(target, filepath.Join(dir, name)))
	}
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "fifo"), 0600))

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
				path      string
				preserved string
				err       error
			}{
				{"file", "file", nil},
				{"directory", "directory", nil},
				{"file_link", "file_link", nil},
				{"directory_link", "directory_link", nil},
				{"socket_link", "socket_link", nil},
				{"dangling_link", "dangling_link", nil},
				{"loop", "loop", nil},
				{"fifo", "fifo", nil},
				{"file/child", "file", syscall.ENOTDIR},
				{"loop/child", "loop", syscall.ELOOP},
			} {
				t.Run(tc.path, func(t *testing.T) {
					preserved := filepath.Join(dir, tc.preserved)
					before, err := os.Lstat(preserved)
					require.NoError(t, err)
					listener, err := factory.create("unix://" + filepath.Join(dir, tc.path))
					if listener != nil {
						t.Cleanup(func() { _ = listener.Close() })
					}
					assert.Error(t, err)
					assert.Nil(t, listener)
					if tc.err != nil {
						assert.ErrorIs(t, err, tc.err)
					}
					after, err := os.Lstat(preserved)
					require.NoError(t, err)
					assert.True(t, os.SameFile(before, after))
					assert.Equal(t, before.Mode(), after.Mode())
				})
			}
		})
	}
	contents, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(contents))
	actualSocket, err := os.Lstat(socketPath)
	require.NoError(t, err)
	assert.True(t, os.SameFile(socketInfo, actualSocket))
	assert.Equal(t, socketInfo.Mode(), actualSocket.Mode())
}

func TestUnixSocketReplacesSocket(t *testing.T) {
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
			path := filepath.Join(socketTempDir(t), "socket")
			original, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
			require.NoError(t, err)
			original.SetUnlinkOnClose(false)
			t.Cleanup(func() { _ = original.Close() })
			before, err := os.Lstat(path)
			require.NoError(t, err)
			listener, err := factory.create("unix://" + path)
			require.NoError(t, err)
			t.Cleanup(func() { _ = listener.Close() })
			after, err := os.Lstat(path)
			require.NoError(t, err)
			assert.Equal(t, os.ModeSocket, after.Mode().Type())
			assert.False(t, os.SameFile(before, after))
			require.NoError(t, listener.Close())
			_, err = os.Lstat(path)
			assert.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestUnixSocketMissingParent(t *testing.T) {
	t.Parallel()
	parent := filepath.Join(socketTempDir(t), "missing")
	uid, gid := os.Geteuid(), os.Getegid()
	listener, err := tcplisten.CreateListenerWithOptions("unix://"+filepath.Join(parent, "socket"), &tcplisten.UnixSocketOptions{Mode: "0777", UID: &uid, GID: &gid})
	if listener != nil {
		t.Cleanup(func() { _ = listener.Close() })
	}
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, listener)
	_, err = os.Lstat(parent)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestUnixSocketChownFailureCleanup(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Root can change socket ownership")
	}
	dir := socketTempDir(t)
	probe := filepath.Join(dir, "probe")
	require.NoError(t, os.WriteFile(probe, nil, 0600))
	if err := os.Chown(probe, 0, -1); err == nil {
		t.Skip("Process can change file ownership to root")
	} else {
		require.ErrorIs(t, err, os.ErrPermission)
	}
	uid := 0
	paths := []string{filepath.Join(dir, "socket")}
	if runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		t.Chdir(dir)
		paths = append(paths, "@socket")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			listener, err := tcplisten.CreateListenerWithOptions("unix://"+path, &tcplisten.UnixSocketOptions{Mode: "0600", UID: &uid})
			if listener != nil {
				t.Cleanup(func() { _ = listener.Close() })
			}
			assert.Nil(t, listener)
			require.ErrorIs(t, err, os.ErrPermission)
			var pathErr *os.PathError
			require.ErrorAs(t, err, &pathErr)
			assert.Equal(t, "chown", pathErr.Op)
			_, err = os.Lstat(path)
			assert.ErrorIs(t, err, os.ErrNotExist)
			replacement, err := net.Listen("unix", path)
			require.NoError(t, err)
			t.Cleanup(func() { _ = replacement.Close() })
			require.NoError(t, replacement.Close())
		})
	}
}
