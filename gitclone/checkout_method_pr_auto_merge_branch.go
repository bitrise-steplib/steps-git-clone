package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// PRMergeRefParams are parameters to check out a Merge/Pull Request's merge ref (the result of merging the 2 branches)
// When available, the merge ref is created by the git server and passed in the webhook.
// Using a merge ref is preferred over a manual merge because we can shallow-fetch the merge ref only.
type PRMergeRefParams struct {
	// MergeRef contains the changes pre-merged by the git provider (eg. pull/7/merge)
	MergeRef string
	// HeadRef is the head of the PR branch (eg. pull/7/head)
	HeadRef string
}

// NewPRMergeRefParams validates and returns a new PRMergeRefParams
func NewPRMergeRefParams(mergeRef, headRef string) (*PRMergeRefParams, error) {
	if strings.TrimSpace(mergeRef) == "" {
		return nil, NewParameterValidationError("Can't checkout PR: no merge ref specified")
	}
	if strings.TrimSpace(headRef) == "" {
		return nil, NewParameterValidationError("Can't checkout PR: no head ref specified")
	}

	return &PRMergeRefParams{
		MergeRef: mergeRef,
		HeadRef:  headRef,
	}, nil
}

type checkoutPRMergeRef struct {
	params PRMergeRefParams
}

func (c checkoutPRMergeRef) do(gitCmd git.Git, fetchOpts fetchOptions, fallback fallbackRetry) error {
	// https://git-scm.com/book/en/v2/Git-Internals-The-Refspec
	refSpec := fmt.Sprintf("%s:%s", c.remoteMergeRef(), c.localMergeRef())

	// $ git fetch origin refs/remotes/pull/7/merge:refs/pull/7/merge
	err := fetch(gitCmd, originRemoteName, refSpec, fetchOpts)
	if err != nil {
		return err
	}

	// Also fetch the PR head ref because the step exports outputs based on the PR head commit (see output.go)
	// $ git fetch origin refs/remotes/pull/7/head:refs/pull/7/head
	err = c.fetchPRHeadRef(gitCmd, fetchOpts)
	if err != nil {
		return err
	}

	// $ git checkout refs/remotes/pull/7/merge
	err = checkoutWithCustomRetry(gitCmd, c.localMergeRef(), nil)
	if err != nil {
		return err
	}

	return nil
}

func (c checkoutPRMergeRef) getBuildTriggerRef() string {
	return c.localHeadRef()
}

func (c checkoutPRMergeRef) localMergeRef() string {
	return fmt.Sprintf("refs/remotes/%s", c.params.MergeRef)
}

func (c checkoutPRMergeRef) remoteMergeRef() string {
	return fmt.Sprintf("refs/%s", c.params.MergeRef)
}

func (c checkoutPRMergeRef) localHeadRef() string {
	return fmt.Sprintf("refs/remotes/%s", c.params.HeadRef)
}

func (c checkoutPRMergeRef) remoteHeadRef() string {
	return fmt.Sprintf("refs/%s", c.params.HeadRef)
}

func (c checkoutPRMergeRef) fetchPRHeadRef(gitCmd git.Git, fetchOpts fetchOptions) error {
	refSpec := fmt.Sprintf("%s:%s", c.remoteHeadRef(), c.localHeadRef())
	return fetch(gitCmd, originRemoteName, refSpec, fetchOpts)
}
