// Command validate_any_usage enforces the any usage allowlist for public API surfaces.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"colonycore/internal/validation"
)

const (
	defaultAllowlistPath = "internal/ci/any_allowlist.json"
	defaultRoots         = "pkg/pluginapi,pkg/datasetapi,pkg/domain,internal/core,internal/adapters/datasets,internal/infra/blob,internal/infra/persistence,plugins"
)

var (
	exitFunc     = os.Exit
	getwd        = os.Getwd
	validateFunc = validation.ValidateAnyUsageFromFile
)

// main is the program entry point; it invokes run with command-line arguments and the configured validation function, then exits with the returned status code.
func main() {
	exitFunc(run(os.Args, os.Stderr, validateFunc))
}

// run parses command-line flags, invokes the any-usage validator, writes any errors
// or violations to stderr, and returns an exit code.
//
// The args slice should contain the program name followed by flags. The validate
// function is invoked with the allowlist path, the current working directory, and
// the list of root packages to scan. The function returns 0 on success (no
// violations) and 1 on any error, invalid input, or when violations are found.
func run(args []string, stderr io.Writer, validate func(string, string, []string) ([]validation.Error, error)) int {
	if len(args) == 0 {
		return 1
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	allowlist := flags.String("allowlist", defaultAllowlistPath, "path to any usage allowlist")
	rootsFlag := flags.String("roots", defaultRoots, "comma-separated roots to scan")
	if err := flags.Parse(args[1:]); err != nil {
		return 1
	}

	roots := splitRoots(*rootsFlag)
	if len(roots) == 0 {
		_, _ = fmt.Fprintln(stderr, "no roots provided for any usage validation")
		return 1
	}
	baseDir, err := getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
		return 1
	}

	violations, err := validate(*allowlist, baseDir, roots)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "any usage guard failed: %v\n", err)
		return 1
	}
	if len(violations) > 0 {
		_, _ = fmt.Fprintf(stderr, "Found %d disallowed any usages:\n\n", len(violations))
		for _, violation := range violations {
			if _, writeErr := fmt.Fprintf(stderr, "%s:%d\n", violation.File, violation.Line); writeErr != nil {
				return 1
			}
			if violation.Message != "" {
				if _, writeErr := fmt.Fprintf(stderr, "  %s\n", violation.Message); writeErr != nil {
					return 1
				}
			}
			if violation.Code != "" {
				if _, writeErr := fmt.Fprintf(stderr, "  Code: %s\n", violation.Code); writeErr != nil {
					return 1
				}
			}
			if _, writeErr := fmt.Fprintln(stderr); writeErr != nil {
				return 1
			}
		}
		return 1
	}
	return 0
}

// splitRoots parses a comma-separated list of roots and returns a slice of non-empty, trimmed entries.
// It trims surrounding whitespace from the input and each entry; if the input is empty or contains only whitespace, it returns nil.
func splitRoots(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		out = append(out, entry)
	}
	return out
}