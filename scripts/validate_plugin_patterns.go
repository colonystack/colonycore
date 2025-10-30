// validate_plugin_patterns.go provides compile-time validation of plugin code
// to ensure adherence to hexagonal architecture and contextual accessor patterns.
package main

import (
	"fmt"
	"io"
	"os"

	"colonycore/internal/validation"
)

func main() {
	os.Exit(run(os.Args, os.Stderr, validation.ValidatePluginDirectory))
}

func run(args []string, stderr io.Writer, validate func(string) []validation.Error) int {
	if len(args) < 2 {
		progName := "validate_plugin_patterns"
		if len(args) > 0 {
			progName = args[0]
		}
		if _, err := fmt.Fprintf(stderr, "Usage: %s <plugin-directory>\n", progName); err != nil {
			return 1
		}
		return 1
	}

	pluginDir := args[1]
	errors := validate(pluginDir)

	if len(errors) > 0 {
		if _, err := fmt.Fprintf(stderr, "❌ Found %d hexagonal architecture violations:\n\n", len(errors)); err != nil {
			return 1
		}
		for _, err := range errors {
			if _, writeErr := fmt.Fprintf(stderr, "🚨 %s:%d\n", err.File, err.Line); writeErr != nil {
				return 1
			}
			if _, writeErr := fmt.Fprintf(stderr, "   %s\n", err.Message); writeErr != nil {
				return 1
			}
			if _, writeErr := fmt.Fprintf(stderr, "   Code: %s\n\n", err.Code); writeErr != nil {
				return 1
			}
		}
		return 1
	}
	return 0
}
