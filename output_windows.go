package main

import (
	"bytes"
	"io/ioutil"
	"os"
)

func withFileOutput(path string, fn writeFunc) error {
	var mode os.FileMode = 0644

	// Try to re-use mode from an existing file
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}

	var buf bytes.Buffer

	if err := fn(&buf); err != nil {
		return err
	}

	// As of August 2021 the github.com/google/renameio package does not
	// support Windows and falls back to using ioutil.WriteFile.
	return ioutil.WriteFile(path, buf.Bytes(), mode)
}
