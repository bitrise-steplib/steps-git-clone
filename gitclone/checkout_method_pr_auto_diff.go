package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// PRDiffFileParams are parameters to check out a Merge/Pull Request if a diff file is available
type PRDiffFileParams struct {
	BaseBranch string
	PRID       uint
}

// NewPRDiffFileParams validates and returns a new PRDiffFile
func NewPRDiffFileParams(baseBranch string, PRID uint) (*PRDiffFileParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR diff file based checkout strategy can not be used, base branch specified")
	}

	return &PRDiffFileParams{
		BaseBranch: baseBranch,
		PRID:       PRID,
	}, nil
}

//
// checkoutPRDiffFile
type checkoutPRDiffFile struct {
	baseBranch, patch string
}

func (c checkoutPRDiffFile) do(gitCmd git.Git, fetchOptions fetchOptions) error {
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
