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
		fmt.Fprintf(stderr, "Usage: %s <plugin-directory>\n", args[0])
		return 1
	}

	pluginDir := args[1]
	errors := validate(pluginDir)

	if len(errors) > 0 {
		fmt.Fprintf(stderr, "❌ Found %d hexagonal architecture violations:\n\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(stderr, "🚨 %s:%d\n", err.File, err.Line)
			fmt.Fprintf(stderr, "   %s\n", err.Message)
			fmt.Fprintf(stderr, "   Code: %s\n\n", err.Code)
		}
		return 1
	}
	return 0
}
