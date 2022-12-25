package main

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	dto "github.com/prometheus/client_model/go"
)

func newString(value string) *string {
	return &value
}

func newFloat64(value float64) *float64 {
	return &value
}

func withReplacedStdinReader(t *testing.T, content string) {
	origStdinReader := stdinReader

	t.Cleanup(func() {
		stdinReader = origStdinReader
	})

	stdinReader = io.NopCloser(strings.NewReader(content))
}

type fakeReaderWithName struct {
	*strings.Reader
	name string
}

func newFakeReaderWithName(name, content string) *fakeReaderWithName {
	return &fakeReaderWithName{
		name:   name,
		Reader: strings.NewReader(content),
	}
}

func (i *fakeReaderWithName) Name() string {
	return i.name
}

func (i *fakeReaderWithName) Close() error {
	return nil
}

func TestMakeReaderInputWrapper(t *testing.T) {
	for _, tc := range []struct {
		name     string
		r        io.ReadCloser
		wantName string
	}{
		{
			name:     "nil",
			wantName: "<nil>",
		},
		{
			name: "in-memory",
			r:    io.NopCloser(strings.NewReader("content")),
		},
		{
			name:     "file",
			r:        newFakeReaderWithName("testname", "content"),
			wantName: "testname",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := newReaderInputWrapper(tc.r)

			if tc.wantName != "" {
				if diff := cmp.Diff(got.Name(), tc.wantName); diff != "" {
					t.Errorf("reader names difference (-got +want):\n%s", diff)
				}
			}
		})
	}
}

func TestInputWrappersFromPaths(t *testing.T) {
	tmpdir := t.TempDir()

	fileA := filepath.Join(tmpdir, "a.txt")
	fileB := filepath.Join(tmpdir, "b.txt")

	for _, path := range []string{fileA, fileB} {
		if err := os.WriteFile(path, []byte(filepath.Base(path)), 0o644); err != nil {
			t.Error(err)
		}
	}

	for _, tc := range []struct {
		name  string
		paths []string
		want  []string
	}{
		{name: "empty"},
		{
			name:  "stdin only",
			paths: []string{"-"},
			want:  []string{"stdin:stdin only"},
		},
		{
			name:  "mixed",
			paths: []string{fileA, "-", fileB},
			want:  []string{"a.txt", "stdin:mixed", "b.txt"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withReplacedStdinReader(t, "stdin:"+tc.name)

			var got []string

			for _, i := range inputWrappersFromPaths(tc.paths) {
				if err := i.Process(func(r io.Reader) error {
					content, err := io.ReadAll(r)
					if err != nil {
						return err
					}

					got = append(got, string(content))

					return nil
				}); err != nil {
					t.Errorf("%v: %v", i, err)
				}
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("reader contents difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestInputWrappersFromDirs(t *testing.T) {
	tmpdir := t.TempDir()

	fileA := filepath.Join(tmpdir, "a.txt")
	fileB := filepath.Join(tmpdir, "b.txt")
	hiddenA := filepath.Join(tmpdir, ".hidden.txt")

	for _, path := range []string{fileA, fileB, hiddenA} {
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Error(err)
		}
	}

	for _, tc := range []struct {
		name    string
		pattern string
		paths   []string
		want    []string
	}{
		{name: "empty"},
		{
			name:    "no matching",
			pattern: "*.txt",
			paths:   []string{t.TempDir(), t.TempDir(), t.TempDir()},
		},
		{
			name:    "",
			pattern: "[^.]*.txt",
			paths:   []string{tmpdir, t.TempDir()},
			want:    []string{fileA, fileB},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string

			inputs, err := inputWrappersFromDirs(tc.paths, tc.pattern)
			if err != nil {
				t.Errorf("inputWrappersFromDirs() failed: %v", err)
			}

			for _, i := range inputs {
				got = append(got, i.Name())
			}

			if diff := cmp.Diff(got, tc.want, cmpopts.SortSlices(func(a, b string) bool {
				return a < b
			})); diff != "" {
				t.Errorf("discovered file difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestReadMetricFamilies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wrapper inputWrapper
		want    parsedInput
		wantErr *regexp.Regexp
	}{
		{
			name:    "empty",
			wrapper: newReaderInputWrapper(newFakeReaderWithName("testname", "")),
			want: parsedInput{
				name: "testname",
			},
		},
		{
			name: "two metrics",
			wrapper: newReaderInputWrapper(newFakeReaderWithName("metrics",
				"size 444\n# HELP weight foo\n# TYPE weight gauge\nweight 111\n")),
			want: parsedInput{
				name: "metrics",
				families: map[string]*dto.MetricFamily{
					"size":   nil,
					"weight": nil,
				},
			},
		},
		{
			name:    "wrong format",
			wrapper: newReaderInputWrapper(newFakeReaderWithName("bad", "x\ny\nz")),
			wantErr: regexp.MustCompile(`^bad:.+: expected float as value\b`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readMetricFamilies(tc.wrapper)

			if tc.wantErr != nil {
				if err == nil || !tc.wantErr.MatchString(err.Error()) {
					t.Errorf("readMetricFamilies() failed with %v, want match for %q", err, tc.wantErr.String())
				}
			} else if err != nil {
				t.Errorf("readMetricFamilies() failed with %v", err)
			}

			if err == nil {
				opts := []cmp.Option{
					cmp.AllowUnexported(parsedInput{}),
					cmpopts.EquateEmpty(),

					// Comparing dto.MetricFamily is hard, so let's ignore it here
					cmpopts.IgnoreTypes(&dto.MetricFamily{}),
				}

				if diff := cmp.Diff(got, tc.want, opts...); diff != "" {
					t.Errorf("readMetricFamilies() difference (-got +want):\n%s", diff)
				}
			}
		})
	}
}
