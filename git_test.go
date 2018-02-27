package main

import "testing"

func TestFetchArg(t *testing.T) {
	for input, expected := range map[string]string{
		"pull/1/merge":  "pull/1/head:pull/1",
		"pull/22/merge": "pull/22/head:pull/22",
	} {
		actual := fetchArg(input)
		if actual != expected {
			t.Errorf("fetchArg(%q), expected %q, actual %q", input, expected, actual)
		}
	}
}

func TestGetRepo(t *testing.T) {
	expected := "github.com/bitrise-samples/git-clone-test"
	for _, input := range []string{
		"https://github.com/bitrise-samples/git-clone-test.git",
		"git@github.com:bitrise-samples/git-clone-test.git",
		"ssh://git@github.com:22/bitrise-samples/git-clone-test.git",
	} {
		actual := getRepo(input)
		if actual != expected {
			t.Errorf("getRepo(%q), expected %q, actual %q", input, expected, actual)
		}
	}
}
