package main

import (
	"testing"
)

func TestFetchArg(t *testing.T) {
	for input, expected := range map[string]string{
		"pull/1/merge":  "refs/pull/1/head:pull/1",
		"pull/22/merge": "refs/pull/22/head:pull/22",
		"pull/224/qux/merge": "refs/pull/224/qux/head:pull/224/qux",
		"pull/22/baz": "refs/heads/pull/22/baz:pull/22/baz",
		"pull/22/merge/foo": "refs/heads/pull/22/merge/foo:pull/22/merge/foo",
		"feature/bar": "refs/heads/feature/bar:feature/bar",
		"feature/qux/baz": "refs/heads/feature/qux/baz:feature/qux/baz",
	} {
		actual := fetchArg(input)
		if actual != expected {
			t.Errorf("fetchArg(%q), expected %q, actual %q", input, expected, actual)
		}
	}
}

func Test_getRepo(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "HTTPS URL",
			url:  "https://github.com/bitrise-samples/git-clone-test.git",
			want: "github.com/bitrise-samples/git-clone-test",
		},
		{
			name: "Long SSH URL",
			url:  "ssh://git@github.com/bitrise-samples/git-clone-test.git",
			want: "github.com/bitrise-samples/git-clone-test",
		},
		{
			name: "Long SSH URL with a specific port",
			url:  "ssh://git@github.com:22/bitrise-samples/git-clone-test.git",
			want: "github.com/bitrise-samples/git-clone-test",
		},
		{
			name: "Short SSH URL",
			url:  "git@github.com:bitrise-samples/git-clone-test.git",
			want: "github.com/bitrise-samples/git-clone-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRepo(tt.url); got != tt.want {
				t.Errorf("getRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}
