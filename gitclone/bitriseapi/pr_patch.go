package bitriseapi

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/filedownloader"
)

type PatchSource interface {
	// GetPRPatch fetches the git patch file of the PR (if available) and returns its local file path
	GetPRPatch() (string, error)
}

func NewPatchSource(buildURL, apiToken string) PatchSource {
	return apiPatchSource{
		buildURL: buildURL,
		apiToken: apiToken,
	}
}

type apiPatchSource struct {
	buildURL string
	apiToken string
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
		return "", fmt.Errorf("could not parse build URL: %v", err)
	}

	if u.Scheme == "file" {
		return filepath.Join(u.Path, "diff.txt"), nil
	}

	diffURL := fmt.Sprintf("%s/diff.txt?api_token=%s", s.buildURL, s.apiToken)
	fileProvider := input.NewFileProvider(filedownloader.New(http.DefaultClient))
	return fileProvider.LocalPath(diffURL)
}
