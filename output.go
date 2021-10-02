// +build !windows

package main

import (
	"bytes"

	"github.com/google/renameio/v2"
)

func withFileOutput(path string, fn writeFunc) error {
	var buf bytes.Buffer

	if err := fn(&buf); err != nil {
		return err
	}

	return renameio.WriteFile(path, buf.Bytes(), 0o644)
}
