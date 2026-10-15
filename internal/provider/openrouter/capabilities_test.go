package openrouter

import (
	"encoding/json"
	"testing"
)

func TestReferenceCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name, params string
		accepts      bool
		min, max     *int
		svg          bool
	}{
		{"required", `{"input_references":{"type":"range","min":1,"max":10},"output_format":{"type":"enum","values":["svg"]}}`, true, intPtr(1), intPtr(10), true},
		{"optional", `{"input_references":{"type":"range","min":0,"max":1}}`, true, intPtr(0), intPtr(1), false},
		{"legacy", `{"input_references":{"type":"boolean"}}`, true, nil, nil, false},
		{"none", `{}`, false, nil, nil, false},
		{"unrelated fractional range", `{"strength":{"type":"range","min":0.1,"max":0.9}}`, false, nil, nil, false},
		{"invalid fractional count", `{"input_references":{"type":"range","min":0.5,"max":1}}`, true, nil, intPtr(1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var params map[string]supportedParameter
			if err := json.Unmarshal([]byte(tc.params), &params); err != nil {
				t.Fatal(err)
			}
			c := capsFrom(params)
			equal := func(a, b *int) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
			if c.AcceptsReferenceImages != tc.accepts || !equal(c.ReferenceMin, tc.min) || !equal(c.ReferenceMax, tc.max) {
				t.Fatalf("caps = %+v", c)
			}
			if tc.svg {
				if len(c.OutputFormats) != 1 || c.OutputFormats[0] != "svg" {
					t.Fatal(c.OutputFormats)
				}
			} else if len(c.OutputFormats) != 0 {
				t.Fatal(c.OutputFormats)
			}
		})
	}
	for _, id := range []string{"recraft/recraft-v4-styles", "recraft/recraft-v4-styles-pro", "recraft/recraft-v4-styles-vector", "recraft/recraft-v4-styles-pro-vector", "recraft/recraft-v4-vector", "other/styles-vector"} {
		c := modelCaps(imageModel{ID: id})
		want := id != "recraft/recraft-v4-vector" && id != "other/styles-vector"
		if (c.ReferenceMinWidth == 256 && c.ReferenceMinHeight == 256 && len(c.ReferenceMediaTypes) == 3) != want {
			t.Fatalf("%s: %+v", id, c)
		}
	}
}
func intPtr(n int) *int { return &n }
