package gitclone

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

//
// PRDiffFileParams are parameters to check out a Merge/Pull Request if a diff file is available
type PRDiffFileParams struct {
	BaseBranch string
	PRID       uint
}

// NewPRDiffFileParams validates and returns a new PRDiffFile
func NewPRDiffFileParams(baseBranch string, PRID uint) (*PRDiffFileParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR diff file based checkout strategy can not be used: no base branch specified")
	}

	return &PRDiffFileParams{
		BaseBranch: baseBranch,
		PRID:       PRID,
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
	getDiffPath(buildURL, apiToken string, PRID int) (string, error)
}

type defaultPatchSource struct{}

func (defaultPatchSource) getDiffPath(buildURL, apiToken string, prID int) (string, error) {
	url := fmt.Sprintf("%s/diff.txt?api_token=%s", buildURL, apiToken)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Can't download diff file, HTTP status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	diffFile, err := ioutil.TempFile("", fmt.Sprintf("%d.diff", prID))
	if err != nil {
		return "", err
	}

	if _, err := diffFile.Write(body); err != nil {
		return "", err
	}
	if err := diffFile.Close(); err != nil {
		return "", err
	}

	return diffFile.Name(), nil
}
