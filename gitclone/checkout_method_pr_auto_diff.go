package gitclone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
)

//
// checkoutPullRequestAutoDiffFile
type checkoutPullRequestAutoDiffFile struct {
	baseBranch, patch string
	// Other
	fetchTraits            fetchTraits
	shouldUpdateSubmodules bool
}

func (c checkoutPullRequestAutoDiffFile) Validate() error {
	if strings.TrimSpace(c.baseBranch) == "" {
		return errors.New("no base branch specified")
	}

	return nil
}

func (c checkoutPullRequestAutoDiffFile) Do(gitCmd git.Git) *step.Error {
	baseBranchRef := newOriginFetchRef(branchRefPrefix + c.baseBranch)
	if err := fetch(gitCmd, c.fetchTraits, baseBranchRef); err != nil {
		return err
	}

	if err := runner.Run(gitCmd.Checkout(c.baseBranch)); err != nil {
		return newStepError(
			"a",
			fmt.Errorf("checkout failed (%s): %v", c.baseBranch, err),
			"aaa",
		)
	}

	if err := runner.Run(gitCmd.Apply(c.patch)); err != nil {
		return newStepError(
			"a",
			fmt.Errorf("can't apply patch (%s): %v", c.patch, err),
			"aaa",
		)
	}

	if c.shouldUpdateSubmodules {
		if err := updateSubmodules(gitCmd); err != nil {
			return err
		}
	}

	return detachHead(gitCmd)
}
