package core

import (
	"testing"

	"colonycore/pkg/datasetapi"
)

func TestCloneColumnsAndFormats(t *testing.T) {
	if cloneColumns(nil) != nil || cloneFormats(nil) != nil {
		t.Fatal("expected nil for nil inputs")
	}
	if cloneColumns([]datasetapi.Column{}) != nil || cloneFormats([]datasetapi.Format{}) != nil {
		t.Fatal("expected nil for empty inputs")
	}

	cols := []datasetapi.Column{{Name: "A"}, {Name: "B"}}
	clonedCols := cloneColumns(cols)
	if len(clonedCols) != len(cols) {
		t.Fatalf("expected %d columns, got %d", len(cols), len(clonedCols))
	}
	cols[0].Name = "mutated"
	if clonedCols[0].Name != "A" {
		t.Fatal("cloneColumns should copy values")
	}

	formatProvider := datasetapi.DefaultFormatProvider{}
	formats := []datasetapi.Format{formatProvider.CSV(), formatProvider.JSON()}
	clonedFormats := cloneFormats(formats)
	if len(clonedFormats) != len(formats) {
		t.Fatalf("expected %d formats, got %d", len(formats), len(clonedFormats))
	}
	formats[0] = "xml"
	if clonedFormats[0] != formatProvider.CSV() {
		t.Fatal("cloneFormats should copy values")
	}
}
