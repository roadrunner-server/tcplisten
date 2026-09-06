//go:build linux || darwin || freebsd || windows

package tcplisten

import (
	"net"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/roadrunner-server/tcplisten"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixSocketOptionsValidateAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "socket")
	for _, tc := range []struct {
		name    string
		address string
		valid   bool
	}{
		{"absolute", "unix://" + path, true},
		{"relative", "unix://socket", true},
		{"at_in_path", "unix://./@socket", true},
		{"leading_at", "unix://@socket", runtime.GOOS != "linux"},
		{"empty", "", false},
		{"empty_path", "unix://", false},
		{"bare_tcp", "127.0.0.1:0", false},
		{"tcp", "tcp://127.0.0.1:0", false},
		{"bare_path", path, false},
		{"unknown_protocol", "udp://127.0.0.1:0", false},
		{"missing_protocol", "://" + path, false},
		{"multiple_separators", "unix://" + path + "://extra", false},
		{"leading_nul", "unix://\x00socket", false},
		{"embedded_nul", "unix://" + path + "\x00extra", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var absent *tcplisten.UnixSocketOptions
			require.NoError(t, absent.Validate(tc.address))

			options := &tcplisten.UnixSocketOptions{}
			if tc.valid && runtime.GOOS != "windows" {
				require.NoError(t, options.Validate(tc.address))
				return
			}
			require.Error(t, options.Validate(tc.address))
			listener, err := tcplisten.CreateListenerWithOptions(tc.address, options)
			if listener != nil {
				t.Cleanup(func() { _ = listener.Close() })
			}
			assert.Error(t, err)
			assert.Nil(t, listener)
		})
	}
}

func TestUnixSocketOptionsValidateMode(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		valid bool
	}{
		{"", true},
		{"0000", true},
		{"0123", true},
		{"0600", true},
		{"0660", true},
		{"0777", true},
		{"0", false},
		{"432", false}, // Viper can convert bare YAML 0660 to decimal text.
		{"600", false},
		{"660", false},
		{"00600", false},
		{"0o600", false},
		{"-600", false},
		{"+600", false},
		{"0800", false},
		{"0690", false},
		{"0608", false},
		{"1000", false},
		{"1600", false},
		{"2600", false},
		{"4600", false},
		{"7777", false},
		{" 0600", false},
		{"0600 ", false},
		{"0600\n", false},
		{"060\x00", false},
	} {
		t.Run(strconv.Quote(tc.mode), func(t *testing.T) {
			options := &tcplisten.UnixSocketOptions{Mode: tc.mode}
			err := options.Validate("unix://socket")
			if tc.valid && runtime.GOOS != "windows" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestUnixSocketOptionsValidateIDs(t *testing.T) {
	for _, field := range []string{"uid", "gid"} {
		t.Run(field, func(t *testing.T) {
			for _, value := range []int64{-1 << 63, -2, -1, 0, 1, 1<<31 - 1, 1 << 31, 1<<32 - 2, 1<<32 - 1, 1 << 32, 1<<63 - 1} {
				t.Run(strconv.FormatInt(value, 10), func(t *testing.T) {
					id := int(value)
					if int64(id) != value {
						t.Skip("ID is not representable as int on this platform")
					}
					options := &tcplisten.UnixSocketOptions{}
					if field == "uid" {
						options.UID = &id
					} else {
						options.GID = &id
					}
					err := options.Validate("unix://socket")
					if value >= 0 && value < 1<<32-1 && runtime.GOOS != "windows" {
						require.NoError(t, err)
					} else {
						require.Error(t, err)
					}
				})
			}
		})
	}
}

func TestCreateListenerTCP(t *testing.T) {
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
			for _, address := range []string{":0", "127.0.0.1:0", "tcp://:0", "tcp://127.0.0.1:0"} {
				t.Run(address, func(t *testing.T) {
					listener, err := factory.create(address)
					require.NoError(t, err)
					t.Cleanup(func() { _ = listener.Close() })
					addr, ok := listener.Addr().(*net.TCPAddr)
					require.True(t, ok)
					require.Positive(t, addr.Port)
					conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(addr.Port)), time.Second)
					require.NoError(t, err)
					require.NoError(t, conn.Close())
				})
			}
		})
	}
}

func TestCreateListenerTCPRejectsUnixOptions(t *testing.T) {
	id := 0
	for _, tc := range []struct {
		name    string
		options *tcplisten.UnixSocketOptions
	}{
		{"empty", &tcplisten.UnixSocketOptions{}},
		{"mode", &tcplisten.UnixSocketOptions{Mode: "0600"}},
		{"uid", &tcplisten.UnixSocketOptions{UID: &id}},
		{"gid", &tcplisten.UnixSocketOptions{GID: &id}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, address := range []string{"127.0.0.1:0", "tcp://127.0.0.1:0"} {
				require.Error(t, tc.options.Validate(address))
				listener, err := tcplisten.CreateListenerWithOptions(address, tc.options)
				if listener != nil {
					t.Cleanup(func() { _ = listener.Close() })
				}
				assert.Error(t, err, address)
				assert.Nil(t, listener, address)
			}
		})
	}
}
