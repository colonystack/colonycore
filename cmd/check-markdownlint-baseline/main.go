// Package main runs the markdownlint baseline CLI.
package main

import (
	"io"
	"os"

	markdownlintbaseline "colonycore/scripts/check_markdownlint_baseline"
)

func main() {
	os.Exit(run(os.Args, os.Stderr, os.Stdin))
}

func run(args []string, stderr io.Writer, stdin io.Reader) int {
	return markdownlintbaseline.Run(args, stderr, stdin)
}
