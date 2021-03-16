package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

const forkRemoteName = "fork"

// PRManualMergeParams are parameters to check out a Merge Request using manual merge
type PRManualMergeParams struct {
	IsFork bool
	// Source
	HeadBranch  string
	MergeArg    string
	HeadRepoURL string // Optional
	// Target
	BaseBranch string
}

//NewPRManualMergeParams validates and returns a new PRManualMergeParams
func NewPRManualMergeParams(isFork bool, headBranch, commit, forkRepoURL, baseBranch string) (*PRManualMergeParams, error) {
	if err := validatePRManualMergeParams(isFork, headBranch, commit, forkRepoURL, baseBranch); err != nil {
		return nil, err
	}

	if isFork {
		remoteForkBranch := fmt.Sprintf("%s/%s", forkRemoteName, headBranch)
		return &PRManualMergeParams{
			IsFork:      isFork,
			HeadBranch:  headBranch,
			MergeArg:    remoteForkBranch,
			HeadRepoURL: forkRepoURL,
			BaseBranch:  baseBranch,
		}, nil
	} else {
		return &PRManualMergeParams{
			IsFork:      isFork,
			HeadBranch:  headBranch,
			MergeArg:    commit,
			HeadRepoURL: "",
			BaseBranch:  baseBranch,
		}, nil
	}
}

func validatePRManualMergeParams(isFork bool, headBranch, commit, forkRepoURL, baseBranch string) error {
	if strings.TrimSpace(headBranch) == "" {
		return NewParameterValidationError("manual PR merge checkout strategy can not be used: no head branch specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return NewParameterValidationError("manual PR merge checkout strategy can not be used: no base branch specified")
	}

	if isFork {
		if strings.TrimSpace(forkRepoURL) == "" {
			return NewParameterValidationError("manual PR merge chekout strategy can not be used: no base repository URL specified")
		}
	} else {
		if strings.TrimSpace(commit) == "" {
			return NewParameterValidationError("manual PR merge checkout strategy can not be used: no head branch commit hash specified")
		}
	}

	return nil
}

type checkoutPRManualMerge struct {
	params PRManualMergeParams
}

func (c checkoutPRManualMerge) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	// Fetch and checkout base (target) branch
	baseBranchRef := branchRefPrefix + c.params.BaseBranch
	if err := fetchInitialBranch(gitCmd, originRemoteName, baseBranchRef, fetchOptions); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	remoteName := originRemoteName
	if c.params.IsFork {
		// Add fork remote
		if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.HeadRepoURL)); err != nil {
			return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.HeadRepoURL, err)
		}

		remoteName = forkRemoteName
	}

	// Fetch and merge
	branchRef := branchRefPrefix + c.params.HeadBranch
	if err := fetch(gitCmd, remoteName, branchRef, fetchOptions); err != nil {
		return nil
	}

	if err := mergeWithCustomRetry(gitCmd, c.params.MergeArg, fallback); err != nil {
		return err
	}

	return detachHead(gitCmd)
}
