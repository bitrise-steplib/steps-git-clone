package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

// PRManualMergeParams are parameters to check out a Merge Request if no merge branch or diff file is avavilable
type PRManualMergeParams struct {
	// Source
	HeadBranch, Commit string
	// Target
	BaseBranch string
}

//NewPRManualMergeParams  validates and returns a new PRManualMergeParams
func NewPRManualMergeParams(headBranch, commit, baseBranch string) (*PRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used, no head branch specified")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used, no head branch commit hash specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used, no base branch specified")
	}

	return &PRManualMergeParams{
		HeadBranch: headBranch,
		Commit:     commit,
		BaseBranch: baseBranch,
	}, nil
}

//
// checkoutPRManualMerge
type checkoutPRManualMerge struct {
	params PRManualMergeParams
}

func (c checkoutPRManualMerge) do(gitCmd git.Git, fetchOptions fetchOptions) error {
	// Fetch and checkout base (target) branch
	baseBranchRef := *newOriginFetchRef(branchRefPrefix + c.params.BaseBranch)
	if err := fetchInitialBranch(gitCmd, baseBranchRef, fetchOptions); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	// Fetch and merge
	headBranchRef := newOriginFetchRef(branchRefPrefix + c.params.HeadBranch)
	if err := fetch(gitCmd, fetchOptions, headBranchRef); err != nil {
		return nil
	}

	var unshallowFunc func(git.Git, error) error
	if !fetchOptions.IsFullDepth() {
		unshallowFunc = simpleUnshallowFunc
	}

	if err := mergeWithCustomRetry(gitCmd, c.params.Commit, unshallowFunc); err != nil {
		return err
	}

	return detachHead(gitCmd)
}

// ForkPRManualMergeParams are parameters to check out a Pull Request if no merge branch or diff file is available
type ForkPRManualMergeParams struct {
	// Source
	HeadBranch, HeadRepoURL string
	// Target
	BaseBranch string
}

// NewForkPRManualMergeParams validates and returns a new ForkPRManualMergeParams
func NewForkPRManualMergeParams(headBranch, forkRepoURL, baseBranch string) (*ForkPRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge checkout strategy can not be used, no head branch specified")
	}
	if strings.TrimSpace(forkRepoURL) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge chekout strategy can not be used, no base repository URL specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge checkout strategy can not be used, no base branch specified")
	}

	return &ForkPRManualMergeParams{
		HeadBranch:  headBranch,
		HeadRepoURL: forkRepoURL,
		BaseBranch:  baseBranch,
	}, nil
}

//
// checkoutForkPRManualMerge
type checkoutForkPRManualMerge struct {
	params ForkPRManualMergeParams
}

func (c checkoutForkPRManualMerge) do(gitCmd git.Git, fetchOptions fetchOptions) error {
	// Fetch and checkout base branch
	baseBranchRef := *newOriginFetchRef(branchRefPrefix + c.params.BaseBranch)
	if err := fetchInitialBranch(gitCmd, baseBranchRef, fetchOptions); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	const forkRemoteName = "fork"
	// Add fork remote
	if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.HeadRepoURL)); err != nil {
		return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.HeadRepoURL, err)
	}

	// Fetch + merge fork branch
	forkBranchRef := fetchRef{
		remote: forkRemoteName,
		ref:    branchRefPrefix + c.params.HeadBranch,
	}
	remoteForkBranch := fmt.Sprintf("%s/%s", forkRemoteName, c.params.HeadBranch)
	if err := fetch(gitCmd, fetchOptions, &forkBranchRef); err != nil {
		return err
	}

	if err := mergeWithCustomRetry(gitCmd, remoteForkBranch, simpleUnshallowFunc); err != nil {
		return err
	}

	return detachHead(gitCmd)
}
