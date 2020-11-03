package gitclone

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
)

func Test_mapRecommendation(t *testing.T) {
	type args struct {
		tag string
		err error
	}
	tests := []struct {
		name string
		args args
		want step.Recommendation
	}{
		{
			name: "checkout_failed generic error mapping",
			args: args{
				tag: checkoutFailedTag,
				err: errors.New("error: pathspec 'master' did not match any file(s) known to git.")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t checkout your branch.",
				Description: "Our auto-configurator returned the following error:\nerror: pathspec 'master' did not match any file(s) known to git."}),
		},
		{
			name: "fetch_failed generic error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("fetch failed, error: exit status 128")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t fetch your repository.",
				Description: "Our auto-configurator returned the following error:\nfetch failed, error: exit status 128"}),
		},
		{
			name: "fetch_failed permission denied (publickey) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("Permission denied (publickey).")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process, double-check your SSH key and try again."}),
		},
		{
			name: "fetch_failed permission denied (publickey,publickey,gssapi-keyex,gssapi-with-mic,password) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("Permission denied (publickey,gssapi-keyex,gssapi-with-mic,password).")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process, double-check your SSH key and try again."}),
		},
		{
			name: "fetch_failed could not find repository (fatal: repository 'http://localhost/repo.git' not found) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("fatal: repository 'http://localhost/repo.git' not found")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t find a git repository at 'http://localhost/repo.git'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed could not find repository (fatal: 'totally.not.made.up' does not appear to be a git repository) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("fatal: 'totally.not.made.up' does not appear to be a git repository")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t find a git repository at 'totally.not.made.up'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed could not find repository (fatal: https://www.youtube.com/channel/UCh0BVQAUkD3vr3WzmINFO5A/info/refs not valid: is this a git repository?) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("fatal: https://www.youtube.com/channel/UCh0BVQAUkD3vr3WzmINFO5A/info/refs not valid: is this a git repository?")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t find a git repository at 'https://www.youtube.com/channel/UCh0BVQAUkD3vr3WzmINFO5A'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed could not access repository (remote: HTTP Basic: Access denied\nfatal: Authentication failed for 'https://localhost/repo.git') error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("remote: HTTP Basic: Access denied\nfatal: Authentication failed for 'https://localhost/repo.git'")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL."}),
		},
		{
			name: "fetch_failed could not access repository (remote: Invalid username or password(.)\nfatal: Authentication failed for 'https://localhost/repo.git') error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("remote: Invalid username or password(.)\nfatal: Authentication failed for 'https://localhost/repo.git'")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL."}),
		},
		{
			name: "fetch_failed could not access repository (Unauthorized) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("Unauthorized")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL."}),
		},
		{
			name: "fetch_failed could not access repository (Forbidden) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("Forbidden")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL."}),
		},
		{
			name: "fetch_failed could not access repository (fatal: unable to access 'https://git.something.com/group/repo.git/': Failed to connect to git.something.com port 443: Connection timed out) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("fatal: unable to access 'https://git.something.com/group/repo.git/': Failed to connect to git.something.com port 443: Connection timed out")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL."}),
		},
		{
			name: "fetch_failed could not access repository (fatal: unable to access 'https://github.com/group/repo.git)/': The requested URL returned error: 400) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("fatal: unable to access 'https://github.com/group/repo.git)/': The requested URL returned error: 400")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL."}),
		},
		{
			name: "fetch_failed could not connect (ssh: connect to host git.something.com.outer port 22: Connection timed out) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("ssh: connect to host git.something.com port 22: Connection timed out) error mapping")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed could not connect (ssh: connect to host git.something.com.outer port 22: Connection refused) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("ssh: connect to host git.something.com port 22: Connection refused) error mapping")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed could not connect (ssh: connect to host git.something.com.outer port 22: Network is unreachable) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("ssh: connect to host git.something.com port 22: Network is unreachable) error mapping")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed could not connect (ssh: Could not resolve hostname git.something.com: Name or service not known) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("ssh: Could not resolve hostname git.something.com: Name or service not known")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t connect to 'git.something.com'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed could not connect (fatal: unable to access 'https://site.google.com/view/something/': Could not resolve host: site.google.com) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("fatal: unable to access 'https://site.google.com/view/something/': Could not resolve host: site.google.com")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t connect to 'site.google.com'.",
				Description: "Please abort the process, double-check your repository URL and try again."}),
		},
		{
			name: "fetch_failed SAML SSO enforced (ERROR: The `my-company' organization has enabled or enforced SAML SSO) error mapping",
			args: args{
				tag: fetchFailedTag,
				err: errors.New("ERROR: The `my-company' organization has enabled or enforced SAML SSO")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "To access this repository, you need to use SAML SSO.",
				Description: `Please abort the process, update your SSH settings and try again. You can find out more about <a target="_blank" href="https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/authorizing-an-ssh-key-for-use-with-saml-single-sign-on">using SAML SSO in the Github docs</a>.`}),
		},
		{
			name: "update_submodule_failed generic (fatal: no submodule mapping found in .gitmodules for path 'web') error mapping",
			args: args{
				tag: updateSubmodelFailedTag,
				err: errors.New("fatal: no submodule mapping found in .gitmodules for path 'web'")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t fetch your repository.",
				Description: "Our auto-configurator returned the following error:\nfatal: no submodule mapping found in .gitmodules for path 'web'"}),
		},
		{
			name: "update_submodule_failed (remote: Unauthorized LoginAndPassword(Username for 'https/***): User not found) error mapping",
			args: args{
				tag: updateSubmodelFailedTag,
				err: errors.New("remote: Unauthorized LoginAndPassword(Username for 'https/***): User not found")},
			want: newDetailedErrorRecommendation(Detail{
				Title:       "We couldn’t access your repository.",
				Description: "Please abort the process and try again, by providing the repository with SSH URL."}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapRecommendation(tt.args.tag, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapRecommendation() = %v, want %v", got, tt.want)
			}
		})
	}
}
