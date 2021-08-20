// +build !windows

package main

import (
	"os"

	"github.com/google/renameio"
)

func withFileOutput(path string, fn writeFunc) error {
	// As of August 2021 github.com/google/renameio's renameio.WriteFile
	// function does not respect the umask for newly created files.
	t, err := renameio.TempFile("", path)
	if err != nil {
		return err
	}

	defer t.Cleanup()

	// Try to re-use mode from an existing file.
	if fi, err := os.Stat(path); err == nil {
		mode := fi.Mode().Perm()

		if err := t.Chmod(mode); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := fn(t); err != nil {
		return err
	}

	return t.CloseAtomicallyReplace()
}
