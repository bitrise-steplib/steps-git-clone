package gitclone

import "github.com/bitrise-io/go-utils/command/git"

type MockStrategy struct {
	ref string
}

func (m MockStrategy) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	return nil
}

func (m MockStrategy) getBuildTriggerRef() string {
	return m.ref
}
