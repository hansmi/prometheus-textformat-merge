package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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
				got, err := ioutil.ReadFile(tc.path)
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
