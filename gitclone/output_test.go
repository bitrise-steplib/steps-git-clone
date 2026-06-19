package gitclone

import (
	"testing"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/git"
	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/stretchr/testify/assert"
)

func Test_gitOutputs(t *testing.T) {
	gitFactory, err := git.DefaultFactory(t.TempDir(), command.NewFactory(env.NewRepository()))
	if err != nil {
		t.Fatal(err)
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
					envKey:      "GIT_CLONE_COMMIT_AUTHOR_NAME",
					gitTemplate: gitFactory.Log("%an", "ref/tags/1.0.0"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
					gitTemplate: gitFactory.Log("%ae", "ref/tags/1.0.0"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_HASH",
					gitTemplate: gitFactory.Log("%H", "ref/tags/1.0.0"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
					gitTemplate: gitFactory.Log("%s", "ref/tags/1.0.0"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_MESSAGE_BODY",
					gitTemplate: gitFactory.Log("%b", "ref/tags/1.0.0"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_COMMITTER_NAME",
					gitTemplate: gitFactory.Log("%cn", "ref/tags/1.0.0"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_COMMITTER_EMAIL",
					gitTemplate: gitFactory.Log("%ce", "ref/tags/1.0.0"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_COUNT",
					gitTemplate: gitFactory.RevList("HEAD", "--count"),
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
					envKey:      "GIT_CLONE_COMMIT_AUTHOR_NAME",
					gitTemplate: gitFactory.Log("%an", "ref/pull/14/head"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
					gitTemplate: gitFactory.Log("%ae", "ref/pull/14/head"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_HASH",
					gitTemplate: gitFactory.Log("%H", "ref/pull/14/head"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
					gitTemplate: gitFactory.Log("%s", "ref/pull/14/head"),
				},
				{
					envKey:      "GIT_CLONE_COMMIT_MESSAGE_BODY",
					gitTemplate: gitFactory.Log("%b", "ref/pull/14/head"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckoutStateResult{
				gitRef:     tt.args.gitRef,
				isPR:       tt.args.isPR,
				gitFactory: gitFactory,
			}
			e := NewOutputExporter(log.NewLogger(), command.NewFactory(env.NewRepository()), r)
			assert.Equalf(t, tt.want, e.gitOutputs(tt.args.gitRef, tt.args.isPR), "gitOutputs(%v, %v)", tt.args.gitRef, tt.args.isPR)
		})
	}
}
