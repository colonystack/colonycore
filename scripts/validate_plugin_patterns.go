// validate_plugin_patterns.go provides compile-time validation of plugin code
// to ensure adherence to hexagonal architecture and contextual accessor patterns.
package main

import (
	"fmt"
	"os"

	"colonycore/internal/validation"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <plugin-directory>\n", os.Args[0])
		os.Exit(1)
	}

	pluginDir := os.Args[1]
	errors := validation.ValidatePluginDirectory(pluginDir)

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "❌ Found %d hexagonal architecture violations:\n\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "🚨 %s:%d\n", err.File, err.Line)
			fmt.Fprintf(os.Stderr, "   %s\n", err.Message)
			fmt.Fprintf(os.Stderr, "   Code: %s\n\n", err.Code)
		}
		os.Exit(1)
	}
}
