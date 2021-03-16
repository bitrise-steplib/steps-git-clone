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
	BaseBranch            string
	PRManualMergeStrategy checkoutStrategy
}

// NewPRDiffFileParams validates and returns a new PRDiffFileParams
func NewPRDiffFileParams(
	baseBranch string,
	prManualMergeStrategy checkoutStrategy,
) (*PRDiffFileParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR diff file based checkout strategy can not be used: no base branch specified")
	}

	return &PRDiffFileParams{
		BaseBranch:            baseBranch,
		PRManualMergeStrategy: prManualMergeStrategy,
	}, nil
}

// checkoutPRDiffFile
type checkoutPRDiffFile struct {
	params    PRDiffFileParams
	patchFile string
}

func (c checkoutPRDiffFile) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	baseBranchRef := branchRefPrefix + c.params.BaseBranch
	if err := fetch(gitCmd, originRemoteName, baseBranchRef, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.BaseBranch, fallback); err != nil {
		return err
	}

	if err := runner.Run(gitCmd.Apply(c.patchFile)); err != nil {
		log.Warnf("Could not apply patch (%s): %v", c.patchFile, err)
		log.Warnf("Falling back to manual merge...")

		if err := c.params.PRManualMergeStrategy.do(gitCmd, fetchOptions, fallback); err != nil {
			return fmt.Errorf("Applying patch (%s) failed: %v", c.patchFile, err)
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
