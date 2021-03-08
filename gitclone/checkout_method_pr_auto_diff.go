package gitclone

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-steplib/bitrise-step-export-universal-apk/filedownloader"
)

//
// PRDiffFileParams are parameters to check out a Merge/Pull Request if a diff file is available
type PRDiffFileParams struct {
	BaseBranch string
}

// NewPRDiffFileParams validates and returns a new PRDiffFile
func NewPRDiffFileParams(baseBranch string, PRID uint) (*PRDiffFileParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR diff file based checkout strategy can not be used: no base branch specified")
	}

	return &PRDiffFileParams{
		BaseBranch: baseBranch,
	}, nil
}

// checkoutPRDiffFile
type checkoutPRDiffFile struct {
	baseBranch, patchFile string
}

func (c checkoutPRDiffFile) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	baseBranchRef := branchRefPrefix + c.baseBranch
	if err := fetch(gitCmd, defaultRemoteName, &baseBranchRef, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.baseBranch, fallback); err != nil {
		return err
	}

	if err := runner.Run(gitCmd.Apply(c.patchFile)); err != nil {
		return fmt.Errorf("could not apply patch (%s): %v", c.patchFile, err)
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
		return "", fmt.Errorf("Could not parse diff file URL: %v", err)
	}

	if url.Scheme == "file" {
		return filepath.Join(buildURL, "diff.txt"), nil
	}

	diffURL := fmt.Sprintf("%s/diff.txt?api_token=%s", buildURL, apiToken)
	fileProvider := input.NewFileProvider(filedownloader.New(http.DefaultClient))
	return fileProvider.LocalPath(diffURL)
}