package datasetapi

import "testing"

func TestFormatProvider(t *testing.T) {
	t.Run("GetFormatProvider returns default provider", func(t *testing.T) {
		provider := GetFormatProvider()
		if provider == nil {
			t.Fatal("GetFormatProvider should not return nil")
		}
	})

	t.Run("format provider provides all expected formats", func(t *testing.T) {
		provider := GetFormatProvider()

		if provider.JSON() != "json" {
			t.Errorf("Expected JSON format 'json', got '%s'", provider.JSON())
		}

		if provider.CSV() != "csv" {
			t.Errorf("Expected CSV format 'csv', got '%s'", provider.CSV())
		}

		if provider.Parquet() != "parquet" {
			t.Errorf("Expected Parquet format 'parquet', got '%s'", provider.Parquet())
		}

		if provider.PNG() != "png" {
			t.Errorf("Expected PNG format 'png', got '%s'", provider.PNG())
		}

		if provider.HTML() != "html" {
			t.Errorf("Expected HTML format 'html', got '%s'", provider.HTML())
		}
	})
}
