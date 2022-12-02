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
	DestinationBranch string
	// MergeRef contains the changes pre-merged by the git provider (eg. pull/7/merge)
	MergeRef string
}

// NewPRMergeRefParams validates and returns a new PRMergeRefParams
func NewPRMergeRefParams(destBranch, mergeRef string) (*PRMergeRefParams, error) {
	if strings.TrimSpace(destBranch) == "" {
		return nil, NewParameterValidationError("Can't checkout PR: no destination branch specified")
	}
	if strings.TrimSpace(mergeRef) == "" {
		return nil, NewParameterValidationError("Can't checkout PR: no merge ref specified")
	}

	return &PRMergeRefParams{
		DestinationBranch: destBranch,
		MergeRef:          mergeRef,
	}, nil
}

type checkoutPRMergeRef struct {
	params PRMergeRefParams
}

func (c checkoutPRMergeRef) do(gitCmd git.Git, fetchOpts fetchOptions, fallback fallbackRetry) error {
	// https://git-scm.com/book/en/v2/Git-Internals-The-Refspec
	refSpec := fmt.Sprintf("%s:%s", c.remoteRef(), c.localRef())

	// $ git fetch origin refs/remotes/pull/7/merge:refs/pull/7/merge
	err := fetch(gitCmd, originRemoteName, refSpec, fetchOpts)
	if err != nil {
		return err
	}

	// $ git checkout refs/remotes/pull/7/merge
	err = checkoutWithCustomRetry(gitCmd, c.localRef(), nil)
	if err != nil {
		return err
	}

	return nil
}

func (c checkoutPRMergeRef) getBuildTriggerRef() string {
	return c.localRef()
}

func (c checkoutPRMergeRef) localRef() string {
	return fmt.Sprintf("refs/remotes/%s", c.params.MergeRef)
}

func (c checkoutPRMergeRef) remoteRef() string {
	return fmt.Sprintf("refs/%s", c.params.MergeRef)
}
