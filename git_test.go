package main

import "testing"

func TestFetchArg(t *testing.T) {
	for input, expected := range map[string]string{
		"pull/1/merge":   "pull/1/head:pull/1",
		"pull/22/merge":  "pull/22/head:pull/22",
		"pull/333/merge": "pull/333/head:pull/333",
	} {
		actual := fetchArg(input)
		if actual != expected {
			t.Errorf("fetchArg(%q), expected %q, actual %q", input, expected, actual)
		}
	}
}
