package gitclone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

//
// checkoutPullRequestAutoDiffFile
type checkoutPullRequestAutoDiffFile struct {
	baseBranch, patch string
}

func (c checkoutPullRequestAutoDiffFile) Validate() error {
	if strings.TrimSpace(c.baseBranch) == "" {
		return errors.New("no base branch specified")
	}

	return nil
}

func (c checkoutPullRequestAutoDiffFile) Do(gitCmd git.Git, fetchOptions fetchOptions) error {
	baseBranchRef := newOriginFetchRef(branchRefPrefix + c.baseBranch)
	if err := fetch(gitCmd, fetchOptions, baseBranchRef); err != nil {
		return err
	}

	if err := runner.Run(gitCmd.Checkout(c.baseBranch)); err != nil {
		return fmt.Errorf("checkout failed (%s): %v", c.baseBranch, err)
	}

	if err := runner.Run(gitCmd.Apply(c.patch)); err != nil {
		return fmt.Errorf("can't apply patch (%s): %v", c.patch, err)
	}

	return detachHead(gitCmd)
}
