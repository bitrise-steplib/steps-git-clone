package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

// PRManualMergeParams are parameters to check out a Merge Request using manual merge
type PRManualMergeParams struct {
	SourceBranch      string
	SourceMergeArg    string
	SourceRepoURL     string // Optional
	DestinationBranch string
}

//NewPRManualMergeParams validates and returns a new PRManualMergeParams
func NewPRManualMergeParams(sourceBranch, commit, sourceRepoURL, destBranch string) (*PRManualMergeParams, error) {
	if err := validatePRManualMergeParams(sourceBranch, commit, sourceRepoURL, destBranch); err != nil {
		return nil, err
	}

	prManualMergeParams := &PRManualMergeParams{
		SourceBranch:      sourceBranch,
		DestinationBranch: destBranch,
	}

	if sourceRepoURL != "" {
		prManualMergeParams.SourceMergeArg = fmt.Sprintf("%s/%s", forkRemoteName, sourceBranch)
		prManualMergeParams.SourceRepoURL = sourceRepoURL
	} else {
		prManualMergeParams.SourceMergeArg = commit
		prManualMergeParams.SourceRepoURL = ""
	}

	return prManualMergeParams, nil
}

func validatePRManualMergeParams(sourceBranch, commit, sourceRepoURL, destBranch string) error {
	if strings.TrimSpace(sourceBranch) == "" {
		return NewParameterValidationError("manual PR merge checkout strategy can not be used: no source branch specified")
	}

	if strings.TrimSpace(destBranch) == "" {
		return NewParameterValidationError("manual PR merge checkout strategy can not be used: no destination branch specified")
	}

	if strings.TrimSpace(sourceRepoURL) == "" && strings.TrimSpace(commit) == "" {
		return NewParameterValidationError("manual PR merge chekout strategy can not be used: no source repository URL or source branch commit hash specified")
	}

	return nil
}

type checkoutPRManualMerge struct {
	params PRManualMergeParams
}

func (c checkoutPRManualMerge) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	// Fetch and checkout destinations branch
	destBranchRef := refsHeadsPrefix + c.params.DestinationBranch
	if err := fetchInitialBranch(gitCmd, originRemoteName, destBranchRef, fetchOptions); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	var remoteName string
	if c.params.SourceRepoURL != "" {
		remoteName = forkRemoteName

		// Add fork remote
		if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.SourceRepoURL)); err != nil {
			return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.SourceRepoURL, err)
		}

	} else {
		remoteName = originRemoteName
	}

	// Fetch and merge
	sourceBranchRef := refsHeadsPrefix + c.params.SourceBranch
	if err := fetch(gitCmd, remoteName, sourceBranchRef, fetchOptions); err != nil {
		return err
	}

	if err := mergeWithCustomRetry(gitCmd, c.params.SourceMergeArg, fallback); err != nil {
		return err
	}

	return detachHead(gitCmd)
}

func (c checkoutPRManualMerge) getBuildTriggerRef() string {
	return c.params.SourceMergeArg
}
