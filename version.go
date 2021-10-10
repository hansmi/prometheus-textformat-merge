package main

import (
	"fmt"
	"io"
)

// https://goreleaser.com/cookbooks/using-main.version
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func showVersion(w io.Writer) {
	fmt.Fprintf(w, "prometheus-textformat-merge version %s (commit %s, built at %s).\n", version, commit, date)
}
