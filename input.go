package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"golang.org/x/sync/errgroup"
)

const stdinPlaceholder = "-"

var stdinReader io.ReadCloser = os.Stdin

type inputWrapper interface {
	Name() string
	Process(func(io.Reader) error) error
}

// processAndClose invokes the given function, passing the reader as an
// argument, and always closes the reader before returning.
func processAndClose(name string, r io.ReadCloser, fn func(io.Reader) error) error {
	defer r.Close()

	if err := fn(r); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	return nil
}

type readerInputWrapper struct {
	name string
	r    io.ReadCloser
}

var _ inputWrapper = (*readerInputWrapper)(nil)

func newReaderInputWrapper(r io.ReadCloser) *readerInputWrapper {
	var name string

	if rn, ok := r.(interface{ Name() string }); ok {
		name = rn.Name()
	} else {
		name = fmt.Sprint(r)
	}

	return &readerInputWrapper{
		name: name,
		r:    r,
	}
}

func (w *readerInputWrapper) Name() string {
	return w.name
}

func (w *readerInputWrapper) Process(fn func(io.Reader) error) error {
	return processAndClose(w.name, w.r, fn)
}

type fileInputWrapper struct {
	path string
}

var _ inputWrapper = (*fileInputWrapper)(nil)

func (w *fileInputWrapper) Name() string {
	return w.path
}

func (w *fileInputWrapper) Process(fn func(io.Reader) error) error {
	r, err := os.Open(w.path)
	if err != nil {
		return err
	}

	return processAndClose(r.Name(), r, fn)
}

func inputWrappersFromPaths(paths []string) []inputWrapper {
	var result []inputWrapper

	for _, i := range paths {
		var r inputWrapper

		if i == stdinPlaceholder {
			r = newReaderInputWrapper(stdinReader)
		} else {
			r = &fileInputWrapper{path: i}
		}

		result = append(result, r)
	}

	return result
}

func inputWrappersFromDirs(paths []string, pattern string) ([]inputWrapper, error) {
	var result []inputWrapper

	for _, path := range paths {
		entries, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, err
		}

		for _, i := range entries {
			if !i.Mode().IsRegular() {
				continue
			}

			matched, err := filepath.Match(pattern, i.Name())
			if err != nil {
				return nil, err
			}

			if matched {
				result = append(result, &fileInputWrapper{
					path: filepath.Join(path, i.Name()),
				})
			}
		}
	}

	return result, nil
}

func readMetricFamilies(w inputWrapper) (parsedInput, error) {
	var families map[string]*dto.MetricFamily

	if err := w.Process(func(r io.Reader) error {
		var err error
		var parser expfmt.TextParser
		families, err = parser.TextToMetricFamilies(r)
		return err
	}); err != nil {
		return parsedInput{}, err
	}

	return parsedInput{
		name:     w.Name(),
		families: families,
	}, nil
}

type parsedInput struct {
	name     string
	families map[string]*dto.MetricFamily
}

// readInputs parses all inputs, up to GOMAXPROCS concurrently, before sending
// the resulting metric families to the given channel. Input order is preserved.
func readInputs(ctx context.Context, inputs []inputWrapper, parsedCh chan<- parsedInput) error {
	g, ctx := errgroup.WithContext(ctx)

	// Limit number of outstanding readers. The outer channel is used to
	// preserve the input order.
	readers := make(chan (<-chan parsedInput), runtime.GOMAXPROCS(0))

	g.Go(func() error {
		defer close(readers)

		for _, w := range inputs {
			w := w

			r := make(chan parsedInput)

			select {
			case readers <- r:
			case <-ctx.Done():
				return ctx.Err()
			}

			g.Go(func() error {
				defer close(r)

				p, err := readMetricFamilies(w)
				if err != nil {
					return err
				}

				select {
				case r <- p:
				case <-ctx.Done():
					return ctx.Err()
				}

				return nil
			})
		}

		return nil
	})

	g.Go(func() error {
		for r := range readers {
			for p := range r {
				parsedCh <- p
			}
		}

		return nil
	})

	return g.Wait()
}
