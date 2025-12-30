package datasetapi

import "testing"

var benchmarkCloneSink any

var benchmarkAttributes = map[string]any{
	"weight":  50.5,
	"notes":   "baseline",
	"tags":    []string{"alpha", "beta", "gamma"},
	"metrics": []any{1, 2.5, "stable", true},
	"nested": map[string]any{
		"level": 2,
		"flags": []any{true, false, "maybe"},
		"scores": []map[string]any{
			{"name": "baseline", "value": 1},
			{"name": "control", "value": 2, "window": []string{"pre", "post"}},
		},
		"attrs": map[string]any{
			"unit":   "mg",
			"values": []any{1, 2, 3},
		},
	},
}

// BenchmarkDeepCloneAttributes measures deepClone time and allocations when cloning
// the nested benchmarkAttributes graph used by dataset payloads.
func BenchmarkDeepCloneAttributes(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkCloneSink = deepClone(benchmarkAttributes)
	}
}

func BenchmarkExtensionPayloadMap(b *testing.B) {
	payload := NewExtensionPayload(benchmarkAttributes)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkCloneSink = payload.Map()
	}
}
