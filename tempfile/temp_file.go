//go:build !linux
// +build !linux

package tempfile

import (
	"os"
)

func MakeTempFile(dir string) (File, error) {
	f, err := os.CreateTemp(dir, "*")
	if err != nil {
		return nil, err
	}
	// Unlink file so that its deleted as soon as its closed.
	os.Remove(f.Name())
	return f, nil
}
