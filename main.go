package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

var stdoutWriter io.Writer = os.Stdout

type writeFunc func(io.Writer) error

func withOutput(path string, fn writeFunc) error {
	if path == "" {
		return fn(stdoutWriter)
	}

	return withFileOutput(path, fn)
}

type cliFlags struct {
	showInputs      bool
	outputFile      string
	dirs            bool
	dirEntryPattern string
}

func (f *cliFlags) register(fs *flag.FlagSet) {
	fs.BoolVar(&f.showInputs, "show-inputs", false, "Emit comment with paths of input files")
	fs.StringVar(&f.outputFile, "output", "", "Write merged metrics to given file instead of standard output")
	fs.BoolVar(&f.dirs, "dirs", false, "Read metrics from regular files in directories given as command arguments")
	fs.StringVar(&f.dirEntryPattern, "dir-entry-pattern", "[^.]*.prom", "Glob pattern for directory entries")
}

func (f *cliFlags) inputs(fs *flag.FlagSet) ([]inputWrapper, error) {
	if f.dirs {
		return inputWrappersFromDirs(flag.Args(), f.dirEntryPattern)
	}

	var paths []string

	if fs.NArg() == 0 {
		paths = []string{stdinPlaceholder}
	} else {
		paths = fs.Args()
	}

	return inputWrappersFromPaths(paths), nil
}

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()

		fmt.Fprintf(w, "Usage: %s [file...]\n", os.Args[0])
		fmt.Fprintln(w, `
Combine one or multiple Prometheus text format inputs. Metric families sharing
a name must also have the same type. The lexicographically lowest help string
per family is used. The resulting metrics are not validated.

If no input files are given standard input is read. Use "-" as a placeholder to
combine standard input with regular files.

Flags:`)
		flag.PrintDefaults()
	}

	var cf cliFlags

	cf.register(flag.CommandLine)

	flag.Parse()

	inputs, err := cf.inputs(flag.CommandLine)
	if err != nil {
		log.Fatal(err)
	}

	merged, err := readAndMerge(context.Background(), inputs)
	if err != nil {
		log.Fatal(err)
	}

	if err := withOutput(cf.outputFile, func(w io.Writer) error {
		return merged.write(w, cf.showInputs)
	}); err != nil {
		log.Fatalf("Writing output failed: %v", err)
	}
}
