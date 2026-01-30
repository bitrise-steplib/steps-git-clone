package bitriseapi

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/filedownloader"
	"github.com/bitrise-io/go-utils/v2/log"
)

type PatchSource interface {
	// GetPRPatch fetches the git patch file of the PR (if available) and returns its local file path
	GetPRPatch() (string, error)
}

func NewPatchSource(buildURL, apiToken string, logger log.Logger) PatchSource {
	return apiPatchSource{
		buildURL: buildURL,
		apiToken: apiToken,
		logger:   logger,
	}
}

type apiPatchSource struct {
	buildURL string
	apiToken string
	logger   log.Logger
}

func (s apiPatchSource) GetPRPatch() (string, error) {
	if s.buildURL == "" {
		return "", fmt.Errorf("Bitrise build URL is not defined")
	}
	if s.apiToken == "" {
		return "", fmt.Errorf("Bitrise API token is not defined")
	}

	u, err := url.Parse(s.buildURL)
	if err != nil {
		return "", fmt.Errorf("parse build URL: %v", err)
	}

	if u.Scheme == "file" {
		return filepath.Join(u.Path, "diff.txt"), nil
	}

	diffURL := fmt.Sprintf("%s/diff.txt?api_token=%s", s.buildURL, s.apiToken)
	downloader := filedownloader.NewDownloader(s.logger)
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pr_patch")
	if err != nil {
		return "", fmt.Errorf("create temp directory: %v", err)
	}
	diffFilePath := filepath.Join(tempDir, "pr_diff.txt")
	err = downloader.Download(ctx, diffFilePath, diffURL)
	if err != nil {
		return "", fmt.Errorf("download PR patch: %v", err)
	}
	return diffFilePath, nil
}
