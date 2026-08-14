package cmd

import "testing"

func TestNormalizeGranularity(t *testing.T) {
	cases := map[string]string{
		"function":  "function",
		"func":      "function",
		"FUNCTION":  "function",
		" Func ":    "function",
		"goroutine": "goroutine",
		"":          "goroutine",
		"bogus":     "goroutine",
	}
	for in, want := range cases {
		if got := NormalizeGranularity(in); got != want {
			t.Errorf("NormalizeGranularity(%q) = %q, want %q", in, got, want)
		}
	}
}
