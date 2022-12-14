package gitclone

import (
	"testing"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/stretchr/testify/assert"
)

func Test_gitOutputs(t *testing.T) {
	gitCmd, err := git.New(t.TempDir())
	if err != nil {
		t.Fatalf(err.Error())
	}
	type args struct {
		gitRef string
		isPR   bool
	}
	tests := []struct {
		name string
		args args
		want []gitOutput
	}{
		{
			name: "Non-PR build",
			args: args{
				gitRef: "ref/tags/1.0.0",
				isPR:   false,
			},
			want: []gitOutput{
				{
					envKey: "GIT_CLONE_COMMIT_AUTHOR_NAME",
					gitCmd: gitCmd.Log("%an", "ref/tags/1.0.0"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
					gitCmd: gitCmd.Log("%ae", "ref/tags/1.0.0"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_HASH",
					gitCmd: gitCmd.Log("%H", "ref/tags/1.0.0"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
					gitCmd: gitCmd.Log("%s", "ref/tags/1.0.0"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_MESSAGE_BODY",
					gitCmd: gitCmd.Log("%b", "ref/tags/1.0.0"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_COMMITER_NAME",
					gitCmd: gitCmd.Log("%cn", "ref/tags/1.0.0"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_COMMITER_EMAIL",
					gitCmd: gitCmd.Log("%ce", "ref/tags/1.0.0"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_COUNT",
					gitCmd: gitCmd.RevList("HEAD", "--count"),
				},
			},
		},
		{
			name: "PR build",
			args: args{
				gitRef: "ref/pull/14/head",
				isPR:   true,
			},
			want: []gitOutput{
				{
					envKey: "GIT_CLONE_COMMIT_AUTHOR_NAME",
					gitCmd: gitCmd.Log("%an", "ref/pull/14/head"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
					gitCmd: gitCmd.Log("%ae", "ref/pull/14/head"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_HASH",
					gitCmd: gitCmd.Log("%H", "ref/pull/14/head"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
					gitCmd: gitCmd.Log("%s", "ref/pull/14/head"),
				},
				{
					envKey: "GIT_CLONE_COMMIT_MESSAGE_BODY",
					gitCmd: gitCmd.Log("%b", "ref/pull/14/head"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &outputExporter{
				gitCmd: gitCmd,
			}
			assert.Equalf(t, tt.want, e.gitOutputs(tt.args.gitRef, tt.args.isPR), "gitOutputs(%v, %v)", tt.args.gitRef, tt.args.isPR)
		})
	}
}
