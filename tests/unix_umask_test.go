//go:build linux || darwin || freebsd

package tcplisten

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/roadrunner-server/tcplisten"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixSocketUmask(t *testing.T) {
	const env = "TCPLISTEN_TEST_UMASK"
	if value := os.Getenv(env); value != "" {
		dir := socketTempDir(t)
		mask, err := strconv.ParseUint(value, 8, 9)
		require.NoError(t, err)
		// Umask is process-wide. Change it only in the child process.
		previous := syscall.Umask(int(mask))
		defer syscall.Umask(previous)
		for _, mode := range []os.FileMode{0000, 0600, 0660, 0777} {
			t.Run(fmt.Sprintf("%04o", mode), func(t *testing.T) {
				path := filepath.Join(dir, fmt.Sprintf("socket-%04o", mode))
				listener, err := tcplisten.CreateListenerWithOptions("unix://"+path, &tcplisten.UnixSocketOptions{Mode: fmt.Sprintf("%04o", mode)})
				require.NoError(t, err)
				t.Cleanup(func() { _ = listener.Close() })
				info, err := os.Lstat(path)
				require.NoError(t, err)
				assert.Equal(t, os.ModeSocket|mode, info.Mode())
				file := path + "-file"
				require.NoError(t, os.WriteFile(file, nil, 0666)) //nolint:gosec // G306: The test needs mode 0666 to check the process umask.
				info, err = os.Stat(file)
				require.NoError(t, err)
				assert.Equal(t, uint64(0666)&^mask, uint64(info.Mode().Perm()))
			})
		}
		return
	}
	for _, mask := range []string{"0000", "0022", "0077", "0777"} {
		t.Run(mask, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestUnixSocketUmask$", "-test.count=1", "-test.timeout=30s") //nolint:gosec // G204: This command runs the test binary with fixed test flags.
			command.Env = append(os.Environ(), env+"="+mask)
			output, err := command.CombinedOutput()
			require.NoError(t, err, "%s", output)
		})
	}
}
