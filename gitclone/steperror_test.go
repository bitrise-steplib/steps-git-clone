package gitclone

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-steplib/steps-git-clone/errormapper"
)

var mapRecommendationMock func(tag, errMsg string) step.Recommendation

func Test_mapRecommendation(t *testing.T) {
	type args struct {
		tag    string
		errMsg string
	}
	tests := []struct {
		name string
		args args
		want step.Recommendation
	}{
		{
			name: "checkout_failed generic error mapping",
			args: args{
				tag:    checkoutFailedTag,
				errMsg: "error: pathspec 'master' did not match any file(s) known to git.",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t checkout your branch.",
				Description: "Our auto-configurator returned the following error:\nerror: pathspec 'master' did not match any file(s) known to git.",
			}),
		},
		{
			name: "fetch_failed generic error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "fetch failed, error: exit status 128",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t fetch your repository.",
				Description: "Our auto-configurator returned the following error:\nfetch failed, error: exit status 128",
			}),
		},
		{
			name: "fetch_failed permission denied (publickey) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "Permission denied (publickey).",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process, double-check your SSH key and try again.",
			}),
		},
		{
			name: "fetch_failed permission denied (publickey,publickey,gssapi-keyex,gssapi-with-mic,password) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "Permission denied (publickey,gssapi-keyex,gssapi-with-mic,password).",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process, double-check your SSH key and try again.",
			}),
		},
		{
			name: "fetch_failed could not find repository (fatal: repository 'http://localhost/repo.git' not found) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "fatal: repository 'http://localhost/repo.git' not found",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t find a git repository at 'http://localhost/repo.git'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed could not find repository (fatal: 'totally.not.made.up' does not appear to be a git repository) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "fatal: 'totally.not.made.up' does not appear to be a git repository",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t find a git repository at 'totally.not.made.up'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed could not find repository (fatal: https://www.youtube.com/channel/UCh0BVQAUkD3vr3WzmINFO5A/info/refs not valid: is this a git repository?) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "fatal: https://www.youtube.com/channel/UCh0BVQAUkD3vr3WzmINFO5A/info/refs not valid: is this a git repository?",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t find a git repository at 'https://www.youtube.com/channel/UCh0BVQAUkD3vr3WzmINFO5A'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed could not access repository (remote: HTTP Basic: Access denied\nfatal: Authentication failed for 'https://localhost/repo.git') error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "remote: HTTP Basic: Access denied\nfatal: Authentication failed for 'https://localhost/repo.git'",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL.",
			}),
		},
		{
			name: "fetch_failed could not access repository (remote: Invalid username or password(.)\nfatal: Authentication failed for 'https://localhost/repo.git') error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "remote: Invalid username or password(.)\nfatal: Authentication failed for 'https://localhost/repo.git'",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL.",
			}),
		},
		{
			name: "fetch_failed could not access repository (Unauthorized) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "Unauthorized",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL.",
			}),
		},
		{
			name: "fetch_failed could not access repository (Forbidden) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "Forbidden",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL.",
			}),
		},
		{
			name: "fetch_failed could not access repository (fatal: unable to access 'https://git.something.com/group/repo.git/': Failed to connect to git.something.com port 443: Connection timed out) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "fatal: unable to access 'https://git.something.com/group/repo.git/': Failed to connect to git.something.com port 443: Connection timed out",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL.",
			}),
		},
		{
			name: "fetch_failed could not access repository (fatal: unable to access 'https://github.com/group/repo.git)/': The requested URL returned error: 400) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "fatal: unable to access 'https://github.com/group/repo.git)/': The requested URL returned error: 400",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL.",
			}),
		},
		{
			name: "fetch_failed could not connect (ssh: connect to host git.something.com.outer port 22: Connection timed out) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "ssh: connect to host git.something.com port 22: Connection timed out) error mapping",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed could not connect (ssh: connect to host git.something.com.outer port 22: Connection refused) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "ssh: connect to host git.something.com port 22: Connection refused) error mapping",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed could not connect (ssh: connect to host git.something.com.outer port 22: Network is unreachable) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "ssh: connect to host git.something.com port 22: Network is unreachable) error mapping",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed could not connect (ssh: Could not resolve hostname git.something.com: Name or service not known) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "ssh: Could not resolve hostname git.something.com: Name or service not known",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed could not connect (fatal: unable to access 'https://site.google.com/view/something/': Could not resolve host: site.google.com) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "fatal: unable to access 'https://site.google.com/view/something/': Could not resolve host: site.google.com"},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t connect to 'site.google.com'.",
				Description: "Please abort the process, double-check your repository URL and try again.",
			}),
		},
		{
			name: "fetch_failed SAML SSO enforced (ERROR: The `my-company' organization has enabled or enforced SAML SSO) error mapping",
			args: args{
				tag:    fetchFailedTag,
				errMsg: "ERROR: The `my-company' organization has enabled or enforced SAML SSO",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "To access this repository, you need to use SAML SSO.",
				Description: `Please abort the process, update your SSH settings and try again. You can find out more about <a target="_blank" href="https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/authorizing-an-ssh-key-for-use-with-saml-single-sign-on">using SAML SSO in the Github docs</a>.`,
			}),
		},
		{
			name: "update_submodule_failed generic (fatal: no submodule mapping found in .gitmodules for path 'web') error mapping",
			args: args{
				tag:    updateSubmodelFailedTag,
				errMsg: "fatal: no submodule mapping found in .gitmodules for path 'web'",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t fetch your repository.",
				Description: "Our auto-configurator returned the following error:\nfatal: no submodule mapping found in .gitmodules for path 'web'",
			}),
		},
		{
			name: "update_submodule_failed (remote: Unauthorized LoginAndPassword(Username for 'https/***): User not found) error mapping",
			args: args{
				tag:    updateSubmodelFailedTag,
				errMsg: "remote: Unauthorized LoginAndPassword(Username for 'https/***): User not found",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL.",
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapRecommendation(tt.args.tag, tt.args.errMsg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapRecommendation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newStepError(t *testing.T) {
	type args struct {
		tag      string
		err      error
		shortMsg string
	}
	tests := []struct {
		name string
		args args
		want *step.Error
	}{
		{
			name: "newStepError without recommendation",
			args: args{
				tag:      "test_tag",
				err:      errors.New("fatal error"),
				shortMsg: "unknown error",
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "test_tag",
				Err:      errors.New("fatal error"),
				ShortMsg: "unknown error",
			},
		},
		{
			name: "newStepError with recommendation",
			args: args{
				tag:      "fetch_failed",
				err:      errors.New("Permission denied (publickey)"),
				shortMsg: "unknown error",
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "fetch_failed",
				Err:      errors.New("Permission denied (publickey)"),
				ShortMsg: "unknown error",
				Recommendations: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
					Title:       "We couldn’t access your repository.",
					Description: "Please abort the process, double-check your SSH key and try again.",
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newStepError(tt.args.tag, tt.args.err, tt.args.shortMsg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newStepError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newStepErrorWithRecommendations(t *testing.T) {
	type args struct {
		tag             string
		err             error
		shortMsg        string
		recommendations step.Recommendation
	}
	tests := []struct {
		name string
		args args
		want *step.Error
	}{
		{
			name: "newStepErrorWithRecommendations",
			args: args{
				tag:      "test_tag",
				err:      errors.New("fatal error"),
				shortMsg: "unknown error",
				recommendations: step.Recommendation{
					"Test": "Passed",
				},
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "test_tag",
				Err:      errors.New("fatal error"),
				ShortMsg: "unknown error",
				Recommendations: step.Recommendation{
					"Test": "Passed",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newStepErrorWithRecommendations(tt.args.tag, tt.args.err, tt.args.shortMsg, tt.args.recommendations); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newStepErrorWithRecommendations() = %v, want %v", got, tt.want)
			}
		})
	}
}
