package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
)

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

	// TODO: Add flag to write merged content to file

	includeInputs := flag.Bool("s", false, "Emit comment with paths of input files")

	flag.Parse()

	var paths []string

	if flag.NArg() == 0 {
		paths = []string{stdinPlaceholder}
	} else {
		paths = flag.Args()
	}

	merged, err := readAndMerge(context.Background(), inputWrappersFromPaths(paths))
	if err != nil {
		log.Fatal(err)
	}

	if err := merged.write(os.Stdout, *includeInputs); err != nil {
		log.Fatalf("Writing output failed: %v", err)
	}
}
