package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func withReplacedStdoutWriter(t *testing.T) *strings.Builder {
	origStdoutWriter := stdoutWriter

	t.Cleanup(func() {
		stdoutWriter = origStdoutWriter
	})

	buf := &strings.Builder{}

	stdoutWriter = buf

	return buf
}

func TestWithOutput(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		fn         writeFunc
		want       string
		wantStdout string
		checkErr   func(*testing.T, error)
	}{
		{
			name: "stdout",
			fn: func(w io.Writer) error {
				io.WriteString(w, "text\n")
				return nil
			},
			wantStdout: "text\n",
		},
		{
			name: "normal",
			path: filepath.Join(t.TempDir(), "test.txt"),
			fn: func(w io.Writer) error {
				io.WriteString(w, "content\n")
				return nil
			},
			want: "content\n",
		},
		{
			name: "target dir does not exist",
			path: filepath.Join(t.TempDir(), "dir", "missing", "test.txt"),
			fn: func(w io.Writer) error {
				return nil
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil || !os.IsNotExist(err) {
					t.Errorf("Failed with %v, want IsNotExist", err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout := withReplacedStdoutWriter(t)

			err := withOutput(tc.path, tc.fn)

			if tc.checkErr != nil {
				tc.checkErr(t, err)
			} else if err != nil {
				t.Errorf("withOutput() failed with %v", err)
			}

			if err == nil && tc.path != "" {
				got, err := os.ReadFile(tc.path)
				if err != nil {
					t.Errorf("ReadFile() failed: %v", err)
				}

				if diff := cmp.Diff(string(got), tc.want); diff != "" {
					t.Errorf("File content difference (-got +want):\n%s", diff)
				}
			}

			if diff := cmp.Diff(stdout.String(), tc.wantStdout); diff != "" {
				t.Errorf("stdout difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestWithFileOutputPreserveMode(t *testing.T) {
	for _, mode := range []os.FileMode{
		0o644,
		0o600,
		0o755,
	} {
		t.Run(fmt.Sprintf("%04o", mode), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.txt")

			if fh, err := os.Create(path); err != nil {
				t.Fatalf("Create(%q) failed: %v", path, err)
			} else {
				if err := fh.Chmod(mode); err != nil {
					t.Errorf("Chmod() failed: %v", err)
				}

				fh.Close()
			}

			if err := withFileOutput(path, func(w io.Writer) error {
				io.WriteString(w, "content\n")
				return nil
			}); err != nil {
				t.Errorf("withFileOutput() failed: %v", err)
			}

			if fi, err := os.Lstat(path); err != nil {
				t.Errorf("Lstat(%q) failed: %v", path, err)
			} else if runtime.GOOS == "windows" {
				// As of Go 1.19 only the 0200 bit (owner writable) is used
				// (https://pkg.go.dev/os#Chmod).
			} else if got := fi.Mode() & os.ModePerm; got != mode {
				t.Errorf("Got file mode %04o, want %04o", got, mode)
			}
		})
	}
}
