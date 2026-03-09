// Command colony provides operator tooling for repository-maintained checks.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"colonycore/pkg/datasetapi"
)

var exitFunc = os.Exit

func main() {
	exitFunc(cli(os.Args[1:], os.Stdout, os.Stderr))
}

func cli(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printRootUsage(stderr)
		return 2
	}

	switch args[0] {
	case "lint":
		return lintCLI(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printRootUsage(stderr)
		return 2
	}
}

func lintCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printLintUsage(stderr)
		return 2
	}

	switch args[0] {
	case "dataset":
		return lintDatasetCLI(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown lint command %q\n", args[0])
		printLintUsage(stderr)
		return 2
	}
}

func lintDatasetCLI(args []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("colony lint dataset", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	var fileArgs stringListFlag
	flagSet.Var(&fileArgs, "file", "dataset template JSON file or directory (repeatable)")
	if err := flagSet.Parse(args); err != nil {
		return 2
	}

	paths := make([]string, 0, len(fileArgs)+len(flagSet.Args()))
	paths = append(paths, fileArgs...)
	paths = append(paths, flagSet.Args()...)
	if len(paths) == 0 {
		_, _ = fmt.Fprintln(stderr, "colony lint dataset: provide at least one file or directory")
		return 2
	}

	files, err := collectTemplateJSONFiles(paths)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "colony lint dataset: %v\n", err)
		return 1
	}
	if len(files) == 0 {
		_, _ = fmt.Fprintln(stderr, "colony lint dataset: no JSON files found")
		return 1
	}

	failures := 0
	for _, path := range files {
		if err := lintTemplateFile(path); err != nil {
			failures++
			reportLintFailure(stderr, path, err)
			continue
		}
		_, _ = fmt.Fprintf(stdout, "%s: OK\n", path)
	}

	if failures > 0 {
		_, _ = fmt.Fprintf(stderr, "dataset lint failed: %d/%d file(s) invalid\n", failures, len(files))
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "dataset lint passed: %d file(s) validated\n", len(files))
	return 0
}

func lintTemplateFile(path string) error {
	payload, err := os.ReadFile(path) // #nosec G304: local operator-supplied path
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	var descriptor datasetapi.TemplateDescriptor
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&descriptor); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	if err := decoder.Decode(new(struct{})); err != io.EOF {
		return fmt.Errorf("parse JSON: %w", err)
	}
	if err := datasetapi.ValidateTemplateDescriptor(descriptor); err != nil {
		return err
	}
	return nil
}

func collectTemplateJSONFiles(paths []string) ([]string, error) {
	seen := make(map[string]struct{})
	files := make([]string, 0, len(paths))
	for _, input := range paths {
		path := strings.TrimSpace(input)
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if !info.IsDir() {
			if !isJSONFile(path) {
				return nil, fmt.Errorf("%s: expected .json file", path)
			}
			appendUniqueFile(path, &files, seen)
			continue
		}
		if err := filepath.WalkDir(path, func(candidate string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if !isJSONFile(candidate) {
				return nil
			}
			appendUniqueFile(candidate, &files, seen)
			return nil
		}); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	sort.Strings(files)
	return files, nil
}

func appendUniqueFile(path string, files *[]string, seen map[string]struct{}) {
	key := normalizeFileKey(path)
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}
	*files = append(*files, key)
}

func normalizeFileKey(path string) string {
	cleaned := filepath.Clean(path)
	absolute, err := filepath.Abs(cleaned)
	if err != nil {
		return cleaned
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return absolute
	}
	return filepath.Clean(resolved)
}

func reportLintFailure(stderr io.Writer, path string, err error) {
	_, _ = fmt.Fprintf(stderr, "%s: FAIL\n", path)
	var validationErr *datasetapi.TemplateValidationError
	if errors.As(err, &validationErr) {
		for _, issue := range validationErr.Issues {
			_, _ = fmt.Fprintf(stderr, "  %s: %s\n", issue.Field, issue.Message)
		}
		return
	}
	_, _ = fmt.Fprintf(stderr, "  %v\n", err)
}

func isJSONFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".json")
}

func printRootUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  colony lint dataset [--file <path>] <path> ...")
}

func printLintUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  colony lint dataset [--file <path>] <path> ...")
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("value must not be empty")
	}
	*f = append(*f, value)
	return nil
}
