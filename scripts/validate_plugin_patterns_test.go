package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMainWithValidPlugin(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })

	pluginDir := filepath.Join("..", "plugins", "frog")
	os.Args = []string{"validate_plugin_patterns", pluginDir}

	main()
}
