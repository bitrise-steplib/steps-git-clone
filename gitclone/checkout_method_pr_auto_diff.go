package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

// PRDiffFileParams are parameters to check out a Merge/Pull Request (when a diff file is available)
type PRDiffFileParams struct {
	DestinationBranch     string
	PRManualMergeStrategy checkoutStrategy
}

// NewPRDiffFileParams validates and returns a new PRDiffFileParams
func NewPRDiffFileParams(
	destBranch string,
	prManualMergeStrategy checkoutStrategy,
) (*PRDiffFileParams, error) {
	if strings.TrimSpace(destBranch) == "" {
		return nil, NewParameterValidationError("PR diff file based checkout strategy can not be used: no base branch specified")
	}

	return &PRDiffFileParams{
		DestinationBranch:     destBranch,
		PRManualMergeStrategy: prManualMergeStrategy,
	}, nil
}

// checkoutPRDiffFile
type checkoutPRDiffFile struct {
	params    PRDiffFileParams
	patchFile string
}

func (c checkoutPRDiffFile) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	destBranchRef := refsHeadsPrefix + c.params.DestinationBranch
	if err := fetch(gitCmd, originRemoteName, destBranchRef, fetchOptions); err != nil {
		return fmt.Errorf("failed to fetch base branch: %w", err)
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.DestinationBranch, fallback); err != nil {
		return err
	}

	if err := runner.Run(gitCmd.Apply(c.patchFile)); err != nil {
		log.Warnf("Could not apply patch (%s): %v", c.patchFile, err)
		log.Warnf("Falling back to manual merge...")

		if err := c.params.PRManualMergeStrategy.do(gitCmd, fetchOptions, fallback); err != nil {
			return fmt.Errorf("fallback failed for applying patch (%s): %v", c.patchFile, err)
		}

		return nil
	}

	return detachHead(gitCmd)
}

func (c checkoutPRDiffFile) getBuildTriggerRef() string {
	return ""
}
