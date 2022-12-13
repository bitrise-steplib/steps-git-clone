package gitclone

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/errormapper"
	"github.com/bitrise-io/go-steputils/step"
)

func Test_mapRecommendation_submodule_update(t *testing.T) {
	wantGenericDetailedError := errormapper.DetailedError{
		Title: "We couldn’t update your submodules.",
		Description: `You can continue adding your app, but your builds will fail unless you fix the issue later.
Our auto-configurator returned the following error:
fatal: no submodule mapping found in .gitmodules for path 'web'`,
	}

	wantAuthenticationDetailedError := func(errorMsg string) errormapper.DetailedError {
		return errormapper.DetailedError{
			Title: "We couldn’t access one or more of your Git submodules.",
			Description: fmt.Sprintf(`You can try accessing your submodules <a target="_blank" href="https://devcenter.bitrise.io/faq/adding-projects-with-submodules/">using an SSH key</a>. You can continue adding your app, but your builds will fail unless you fix this issue later.
Our auto-configurator returned the following error:
%s`, errorMsg),
		}
	}

	tests := []struct {
		name   string
		errMsg string
		want   step.Recommendation
	}{
		{
			name:   "update_submodule_failed generic (fatal: no submodule mapping found in .gitmodules for path 'web') error mapping",
			errMsg: "fatal: no submodule mapping found in .gitmodules for path 'web'",
			want:   errormapper.NewDetailedErrorRecommendation(wantGenericDetailedError),
		},
		{
			name:   "update_submodule_failed (ERROR: Repository not found.) error mapping",
			errMsg: "ERROR: Repository not found.",
			want:   errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError("ERROR: Repository not found.")),
		},
		{
			name:   "update_submodule_failed (remote: Invalid username or password(.)) error mapping",
			errMsg: "remote: Invalid username or password(.)",
			want:   errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError("remote: Invalid username or password(.)")),
		},
		{
			name:   "update_submodule_failed (Permission denied (publickey).) error mapping",
			errMsg: "Permission denied (publickey).",
			want:   errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError("Permission denied (publickey).")),
		},
		{
			name: "update_submodule_failed (remote: HTTP Basic: Access denied) error mapping",
			errMsg: `remote: HTTP Basic: Access denied
fatal: Authentication failed for 'https://bitrise-io.git/'`,
			want: errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError(`remote: HTTP Basic: Access denied
fatal: Authentication failed for 'https://bitrise-io.git/'`)),
		},
		{
			name:   "update_submodule_failed (Permission denied, please try again.) error mapping",
			errMsg: "Permission denied, please try again.",
			want:   errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError("Permission denied, please try again.")),
		},
		{
			name:   "update_submodule_failed (Unauthorized) error mapping",
			errMsg: "Unauthorized",
			want:   errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError("Unauthorized")),
		},
		{
			name:   "update_submodule_failed (remote: The project you were looking for could not be found.) error mapping",
			errMsg: "remote: The project you were looking for could not be found.",
			want:   errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError("remote: The project you were looking for could not be found.")),
		},
		{
			name:   "update_submodule_failed (remote: Unauthorized LoginAndPassword(Username for 'https/***): User not found) error mapping",
			errMsg: "remote: Unauthorized LoginAndPassword(Username for 'https/***): User not found",
			want:   errormapper.NewDetailedErrorRecommendation(wantAuthenticationDetailedError("remote: Unauthorized LoginAndPassword(Username for 'https/***): User not found")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapDetailedErrorRecommendation(updateSubmoduleFailedTag, tt.errMsg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapRecommendation() = %v, want %v", got, tt.want)
			}
		})
	}
}
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
				errMsg: "error: fatal: /master: '/master' is outside repository",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn’t checkout your branch.",
				Description: "Our auto-configurator returned the following error:\nerror: fatal: /master: '/master' is outside repository",
			}),
		},
		{
			name: "checkout_failed invalid banch error mapping",
			args: args{
				tag:    checkoutFailedTag,
				errMsg: "error: pathspec 'master' did not match any file(s) known to git.",
			},
			want: errormapper.NewDetailedErrorRecommendation(errormapper.DetailedError{
				Title:       "We couldn't find the branch 'master'.",
				Description: "Please choose another branch and try again.",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapDetailedErrorRecommendation(tt.args.tag, tt.args.errMsg); !reflect.DeepEqual(got, tt.want) {
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

func Test_newStepErrorWithBranchRecommendations(t *testing.T) {
	type args struct {
		tag               string
		err               error
		shortMsg          string
		currentBranch     string
		availableBranches []string
	}
	tests := []struct {
		name string
		args args
		want *step.Error
	}{
		{
			name: "newStepErrorWithBranchRecommendations without available branches",
			args: args{
				tag:               "checkout_failed",
				err:               errors.New("Generic error"),
				shortMsg:          "Checkout has failed",
				currentBranch:     "feature1",
				availableBranches: nil,
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "checkout_failed",
				Err:      errors.New("Generic error"),
				ShortMsg: "Checkout has failed",
				Recommendations: step.Recommendation{
					errormapper.DetailedErrorRecKey: errormapper.DetailedError{
						Title:       "We couldn’t checkout your branch.",
						Description: "Our auto-configurator returned the following error:\nGeneric error",
					},
				},
			},
		},
		{
			name: "newStepErrorWithBranchRecommendations with available branches",
			args: args{
				tag:               "checkout_failed",
				err:               errors.New("pathspec 'feature1' did not match any file(s) known to git"),
				shortMsg:          "Checkout has failed",
				currentBranch:     "feature1",
				availableBranches: []string{"master", "develop", "hotfix"},
			},
			want: &step.Error{
				StepID:   "git-clone",
				Tag:      "checkout_failed",
				Err:      errors.New("pathspec 'feature1' did not match any file(s) known to git"),
				ShortMsg: "Checkout has failed",
				Recommendations: step.Recommendation{
					branchRecKey: []string{"master", "develop", "hotfix"},
					errormapper.DetailedErrorRecKey: errormapper.DetailedError{
						Title:       "We couldn't find the branch 'feature1'.",
						Description: "Please choose another branch and try again.",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newStepErrorWithBranchRecommendations(tt.args.tag, tt.args.err, tt.args.shortMsg, tt.args.currentBranch, tt.args.availableBranches); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newStepErrorWithBranchRecommendations() = %v,want %v", got, tt.want)
			}
		})
	}
}
