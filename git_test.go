package main

import (
	"testing"
)

var testCases = []struct {
	input    string
	expected string
}{
	{
		input:    "pull/1/merge",
		expected: "pull/1/head:pull/1",
	},
	{
		input:    "pull/22/merge",
		expected: "pull/22/head:pull/22",
	},
}

func TestFetchArg(t *testing.T) {
	for _, test := range testCases {
		actual := fetchArg(test.input)
		if actual != test.expected {
			t.Errorf("fetchArg(%q), expected %q, actual %q", test.input, test.expected, actual)
		}
	}
}
