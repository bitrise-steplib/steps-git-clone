package gitclone

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/errormapper"
	"github.com/bitrise-io/go-steputils/step"
	"github.com/stretchr/testify/assert"
)

func TestFetchArg(t *testing.T) {
	tests := []struct {
		name           string
		mergeBranchArg string
		wantRemoteRef  string
		wantLocalRef   string
	}{
		{
			name:           "PR ID short",
			mergeBranchArg: "pull/1/merge",
			wantRemoteRef:  "refs/pull/1/head",
			wantLocalRef:   "pull/1",
		},
		{
			name:           "PR ID long",
			mergeBranchArg: "pull/22/merge",
			wantRemoteRef:  "refs/pull/22/head",
			wantLocalRef:   "pull/22",
		},
		{
			name:           "Extra path element prefixed",
			mergeBranchArg: "pull/224/qux/merge",
			wantRemoteRef:  "refs/pull/224/qux/head",
			wantLocalRef:   "pull/224/qux",
		},
		{
			name:           "Alternate suffix",
			mergeBranchArg: "pull/22/baz",
			wantRemoteRef:  "refs/heads/pull/22/baz",
			wantLocalRef:   "pull/22/baz",
		},
		{
			name:           "Extra path element suffficed",
			mergeBranchArg: "pull/22/merge/foo",
			wantRemoteRef:  "refs/heads/pull/22/merge/foo",
			wantLocalRef:   "pull/22/merge/foo",
		},
		{
			name:           "Non GitHub convention, PR ID missing",
			mergeBranchArg: "feature/bar",
			wantRemoteRef:  "refs/heads/feature/bar",
			wantLocalRef:   "feature/bar",
		},
		{
			name:           "Non GitHub convention, PR ID missing, has extra path elemnent",
			mergeBranchArg: "feature/qux/baz",
			wantRemoteRef:  "refs/heads/feature/qux/baz",
			wantLocalRef:   "feature/qux/baz",
		},
	}
	for _, tt := range tests {
		gotRemoteRef, gotLocalRef := fetchArg(tt.mergeBranchArg)

		assert.Equal(t, tt.wantRemoteRef, gotRemoteRef)
		assert.Equal(t, tt.wantLocalRef, gotLocalRef)
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

func Test_parseListBranchesOutput(t *testing.T) {
	tests := []struct {
		name string
		args string
		want map[string][]string
	}{
		{
			name: "single branch",
			args: "upstream/master",
			want: map[string][]string{
				"upstream": {
					"master",
				},
			},
		},
		{
			name: "multiple branches",
			args: `upstream/bitrise-bot-1
  upstream/bitrise-bot-2
  upstream/bitrise-bot-3`,
			want: map[string][]string{
				"upstream": {
					"bitrise-bot-1",
					"bitrise-bot-2",
					"bitrise-bot-3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseListBranchesOutput(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseListBranchesOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleCheckoutError(t *testing.T) {
	type args struct {
		callback func() (map[string][]string, error)
		tag      string
		err      error
		shortMsg string
		branch   string
	}
	tests := []struct {
		name string
		args args
		want *step.Error
	}{
		{
			name: "handleCheckoutError: generic error without branch recommendation",
			args: args{
				callback: func() (map[string][]string, error) { return nil, nil },
				tag:      "fetch_failed",
				err:      errors.New("Something bad happened"),
				shortMsg: "Fetching repository has failed",
				branch:   "",
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "fetch_failed",
				Err:      errors.New("Something bad happened"),
				ShortMsg: "Fetching repository has failed",
				Recommendations: step.Recommendation{
					errormapper.DetailedErrorRecKey: errormapper.DetailedError{
						Title:       "We couldn’t fetch your repository.",
						Description: "Our auto-configurator returned the following error:\nSomething bad happened",
					},
				},
			},
		},
		{
			name: "handleCheckoutError: specific error with branch recommendations for default remote",
			args: args{
				callback: func() (map[string][]string, error) {
					return map[string][]string{
						originRemoteName: {"master", "develop"},
					}, nil
				},
				tag:      "checkout_failed",
				err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				shortMsg: "Checkout has failed",
				branch:   "test",
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "checkout_failed",
				Err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				ShortMsg: "Checkout has failed",
				Recommendations: step.Recommendation{
					branchRecKey: []string{"master", "develop"},
					errormapper.DetailedErrorRecKey: errormapper.DetailedError{
						Title:       "We couldn't find the branch 'test'.",
						Description: "Please choose another branch and try again.",
					},
				},
			},
		},
		{
			name: "handleCheckoutError: specific error without branch recommendations due to error",
			args: args{
				callback: func() (map[string][]string, error) {
					return nil, errors.New("No available branches")
				},
				tag:      "checkout_failed",
				err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				shortMsg: "Checkout has failed",
				branch:   "test",
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "checkout_failed",
				Err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				ShortMsg: "Checkout has failed",
				Recommendations: step.Recommendation{
					errormapper.DetailedErrorRecKey: errormapper.DetailedError{
						Title:       "We couldn't find the branch 'test'.",
						Description: "Please choose another branch and try again.",
					},
				},
			},
		},
		{
			name: "handleCheckoutError: specific error without branch recommendations due correct branch",
			args: args{
				callback: func() (map[string][]string, error) {
					return map[string][]string{
						originRemoteName: {"master", "develop", "test"},
					}, nil
				},
				tag:      "checkout_failed",
				err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				shortMsg: "Checkout has failed",
				branch:   "test",
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "checkout_failed",
				Err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				ShortMsg: "Checkout has failed",
				Recommendations: step.Recommendation{
					errormapper.DetailedErrorRecKey: errormapper.DetailedError{
						Title:       "We couldn't find the branch 'test'.",
						Description: "Please choose another branch and try again.",
					},
				},
			},
		},
		{
			name: "handleCheckoutError: specific error without branch recommendations for default remote",
			args: args{
				callback: func() (map[string][]string, error) {
					return map[string][]string{
						"something": {"master", "develop"},
					}, nil
				},
				tag:      "checkout_failed",
				err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				shortMsg: "Checkout has failed",
				branch:   "test",
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "checkout_failed",
				Err:      errors.New("pathspec 'test' did not match any file(s) known to git"),
				ShortMsg: "Checkout has failed",
				Recommendations: step.Recommendation{
					errormapper.DetailedErrorRecKey: errormapper.DetailedError{
						Title:       "We couldn't find the branch 'test'.",
						Description: "Please choose another branch and try again.",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleCheckoutError(tt.args.callback, tt.args.tag, tt.args.err, tt.args.shortMsg, tt.args.branch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleCheckoutError() = %v, want %v", got, tt.want)
			}
		})
	}
}
