package pluginapi

import "testing"

func TestVersionProvider(t *testing.T) {
	t.Run("GetVersionProvider returns default provider", func(t *testing.T) {
		provider := GetVersionProvider()
		if provider == nil {
			t.Fatal("GetVersionProvider should not return nil")
		}
	})

	t.Run("default version provider returns correct API version", func(t *testing.T) {
		provider := GetVersionProvider()
		version := provider.APIVersion()

		// The API version should be a non-empty string
		if version == "" {
			t.Error("APIVersion should return a non-empty string")
		}

		// Should return expected version
		if version != "v1" {
			t.Errorf("Expected API version 'v1', got '%s'", version)
		}
	})
}
