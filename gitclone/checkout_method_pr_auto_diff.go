package gitclone

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/bitrise-step-export-universal-apk/filedownloader"
)

//
// PRDiffFileParams are parameters to check out a Merge/Pull Request (when a diff file is available)
type PRDiffFileParams struct {
	BaseBranch             string
	PRManualMergeParam     *PRManualMergeParams
	ForkPRManualMergeParam *ForkPRManualMergeParams
}

// NewPRDiffFileParams validates and returns a new PRDiffFileParams
func NewPRDiffFileParams(
	baseBranch string,
	prManualMergeParam *PRManualMergeParams,
	forkPRManualMergeParam *ForkPRManualMergeParams,
) (*PRDiffFileParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR diff file based checkout strategy can not be used: no base branch specified")
	}

	return &PRDiffFileParams{
		BaseBranch:             baseBranch,
		PRManualMergeParam:     prManualMergeParam,
		ForkPRManualMergeParam: forkPRManualMergeParam,
	}, nil
}

// checkoutPRDiffFile
type checkoutPRDiffFile struct {
	params    PRDiffFileParams
	patchFile string
}

func (c checkoutPRDiffFile) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	baseBranchRef := branchRefPrefix + c.params.BaseBranch
	if err := fetch(gitCmd, defaultRemoteName, &baseBranchRef, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.BaseBranch, fallback); err != nil {
		return err
	}

	if err := runner.Run(gitCmd.Apply(c.patchFile)); err != nil {
		log.Warnf("Could not apply patch (%s): %v", c.patchFile, err)
		log.Warnf("Falling back to manual merge...")

		if c.params.PRManualMergeParam != nil {
			headBranchRef := branchRefPrefix + c.params.PRManualMergeParam.HeadBranch
			if err := fetch(gitCmd, defaultRemoteName, &headBranchRef, fetchOptions); err != nil {
				return nil
			}

			if err := mergeWithCustomRetry(gitCmd, c.params.PRManualMergeParam.Commit, fallback); err != nil {
				return err
			}

			return nil
		} else if c.params.ForkPRManualMergeParam != nil {
			const forkRemoteName = "fork"
			// Add fork remote
			if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.ForkPRManualMergeParam.HeadRepoURL)); err != nil {
				return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.ForkPRManualMergeParam.HeadRepoURL, err)
			}

			// Fetch + merge fork branch
			forkBranchRef := branchRefPrefix + c.params.ForkPRManualMergeParam.HeadBranch
			if err := fetch(gitCmd, forkRemoteName, &forkBranchRef, fetchOptions); err != nil {
				return err
			}

			remoteForkBranch := fmt.Sprintf("%s/%s", forkRemoteName, c.params.ForkPRManualMergeParam.HeadBranch)
			if err := mergeWithCustomRetry(gitCmd, remoteForkBranch, fallback); err != nil {
				return err
			}

			return nil
		} else {
			return fmt.Errorf("could not apply patch (%s): %v", c.patchFile, err)
		}
	}

	return detachHead(gitCmd)
}

type patchSource interface {
	getDiffPath(buildURL, apiToken string) (string, error)
}

type defaultPatchSource struct{}

func (defaultPatchSource) getDiffPath(buildURL, apiToken string) (string, error) {
	url, err := url.Parse(buildURL)
	if err != nil {
		return "", fmt.Errorf("could not parse diff file URL: %v", err)
	}

	if url.Scheme == "file" {
		return filepath.Join(url.Path, "diff.txt"), nil
	}

	diffURL := fmt.Sprintf("%s/diff.txt?api_token=%s", buildURL, apiToken)
	fileProvider := input.NewFileProvider(filedownloader.New(http.DefaultClient))
	return fileProvider.LocalPath(diffURL)
}
