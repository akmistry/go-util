//go:build linux
// +build linux

package tempfile

import (
	"os"

	"golang.org/x/sys/unix"
)

func MakeTempFile(dir string) (File, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	fd, err := unix.Open(dir, unix.O_RDWR|unix.O_CLOEXEC|unix.O_TMPFILE|unix.O_EXCL, 0600)
	if err == nil {
		return os.NewFile(uintptr(fd), "<temp file>"), nil
	}

	f, err := os.CreateTemp(dir, "*")
	if err != nil {
		return nil, err
	}
	// Unlink file so that its deleted as soon as its closed.
	os.Remove(f.Name())
	return f, nil
}
