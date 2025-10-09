package datasetapi

import "testing"

// TestAPIStabilityGuard validates the snapshot mechanism catches changes.
func TestAPIStabilityGuard(t *testing.T) {
	t.Run("snapshot mechanism is functional", func(t *testing.T) {
		// This test just validates that the snapshot mechanism is working
		// by ensuring we can generate and validate snapshots
		path := resolveSnapshotPath(t)
		if path == "" {
			t.Fatal("snapshot path resolution failed")
		}

		snapshot, err := currentAPISnapshot(t)
		if err != nil {
			t.Fatalf("snapshot generation failed: %v", err)
		}

		if len(snapshot) == 0 {
			t.Fatal("snapshot should not be empty")
		}

		// Verify that key interface methods are captured in snapshot
		snapshotContent := string(snapshot)
		requiredElements := []string{
			"TYPE DialectProvider interface",
			"TYPE FormatProvider interface",
			"FUNC GetDialectProvider()",
			"FUNC GetFormatProvider()",
		}

		for _, element := range requiredElements {
			if !contains(snapshotContent, element) {
				t.Errorf("snapshot missing required element: %s", element)
			}
		}
	})

	t.Run("interface methods are captured", func(t *testing.T) {
		snapshot, err := currentAPISnapshot(t)
		if err != nil {
			t.Fatalf("snapshot generation failed: %v", err)
		}

		snapshotContent := string(snapshot)

		// Verify that our new interface methods are captured
		dialectMethods := []string{"DSL()", "SQL()"}
		formatMethods := []string{"CSV()", "HTML()", "JSON()", "PNG()", "Parquet()"}

		for _, method := range dialectMethods {
			if !contains(snapshotContent, method) {
				t.Errorf("DialectProvider method %s not captured in snapshot", method)
			}
		}

		for _, method := range formatMethods {
			if !contains(snapshotContent, method) {
				t.Errorf("FormatProvider method %s not captured in snapshot", method)
			}
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
