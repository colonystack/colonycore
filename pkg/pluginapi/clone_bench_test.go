package pluginapi

import "testing"

var benchmarkCloneSink any

var benchmarkPayload = map[string]any{
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

func BenchmarkCloneValueNested(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkCloneSink = cloneValue(benchmarkPayload)
	}
}

func BenchmarkObjectPayloadMap(b *testing.B) {
	payload := NewObjectPayload(benchmarkPayload)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkCloneSink = payload.Map()
	}
}

func BenchmarkExtensionSetRaw(b *testing.B) {
	raw := map[string]map[string]map[string]any{
		"hook.alpha": {
			"plugin.alpha": benchmarkPayload,
		},
		"hook.beta": {
			"plugin.beta": {"flag": true},
		},
	}
	set := NewExtensionSet(raw)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkCloneSink = set.Raw()
	}
}
