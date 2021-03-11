package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

const forkRemoteName = "fork"

//
// PRManualMergeParams are parameters to check out a Merge Request using manual merge
type PRManualMergeParams struct {
	// Source
	HeadBranch, Commit string
	// Target
	BaseBranch string
}

//NewPRManualMergeParams validates and returns a new PRManualMergeParams
func NewPRManualMergeParams(headBranch, commit, baseBranch string) (*PRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used: no head branch specified")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used: no head branch commit hash specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used: no base branch specified")
	}

	return &PRManualMergeParams{
		HeadBranch: headBranch,
		Commit:     commit,
		BaseBranch: baseBranch,
	}, nil
}

//
// ForkPRManualMergeParams are parameters to check out a Pull Request using manual merge
type ForkPRManualMergeParams struct {
	// Source
	HeadBranch, HeadRepoURL string
	// Target
	BaseBranch string
}

// NewForkPRManualMergeParams validates and returns a new ForkPRManualMergeParams
func NewForkPRManualMergeParams(headBranch, forkRepoURL, baseBranch string) (*ForkPRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge checkout strategy can not be used: no head branch specified")
	}
	if strings.TrimSpace(forkRepoURL) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge chekout strategy can not be used: no base repository URL specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge checkout strategy can not be used: no base branch specified")
	}

	return &ForkPRManualMergeParams{
		HeadBranch:  headBranch,
		HeadRepoURL: forkRepoURL,
		BaseBranch:  baseBranch,
	}, nil
}

// checkoutManualMergeParams are parameters to check out a MR/PR using manual merge
type checkoutManualMergeParams struct {
	// Source
	MergeArg    string
	HeadBranch  string
	HeadRepoURL string // Optional
	// Target
	BaseBranch string
}

func newCheckoutManualMergeMR(params PRManualMergeParams) checkoutManualMergeParams {
	return checkoutManualMergeParams{
		HeadBranch:  params.HeadBranch,
		HeadRepoURL: "",
		BaseBranch:  params.BaseBranch,
		MergeArg:    params.Commit,
	}
}

func newCheckoutManualMergePR(params ForkPRManualMergeParams) checkoutManualMergeParams {
	remoteForkBranch := fmt.Sprintf("%s/%s", forkRemoteName, params.HeadBranch)
	return checkoutManualMergeParams{
		HeadBranch:  params.HeadBranch,
		HeadRepoURL: params.HeadRepoURL,
		BaseBranch:  params.BaseBranch,
		MergeArg:    remoteForkBranch,
	}
}

func (c checkoutManualMergeParams) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	baseBranchRef := branchRefPrefix + c.BaseBranch
	if err := fetchInitialBranch(gitCmd, defaultRemoteName, baseBranchRef, fetchOptions); err != nil {
		return err
	}

	if err := c.manualMerge(gitCmd, fetchOptions, fallback); err != nil {
		return err
	}

	return detachHead(gitCmd)
}

func (c checkoutManualMergeParams) manualMerge(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	remote := defaultRemoteName
	if c.HeadRepoURL != "" {
		remote = forkRemoteName
		// Add fork remote
		if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.HeadRepoURL)); err != nil {
			return fmt.Errorf("adding remote fork repository failed (%s): %v", c.HeadRepoURL, err)
		}
	}

	branchRef := branchRefPrefix + c.HeadBranch
	if err := fetch(gitCmd, remote, branchRef, fetchOptions); err != nil {
		return err
	}

	if err := mergeWithCustomRetry(gitCmd, c.MergeArg, fallback); err != nil {
		return err
	}

	return nil
}
