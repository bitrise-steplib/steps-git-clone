package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type fallbackRetry interface {
	do(gitCmd git.Git, oErr error) error
}

type simpleUnshallow struct{}

func (s simpleUnshallow) do(gitCmd git.Git, oErr error) error {
	log.Warnf("Checkout failed, error: %v\nUnshallow...", oErr)

	if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}

	return nil
}

type resetUnshallow struct{}

func (r resetUnshallow) do(gitCmd git.Git, oErr error) error {
	log.Warnf("Merge failed: %v\nReset repository, then unshallow...", oErr)

	if err := resetRepo(gitCmd); err != nil {
		return fmt.Errorf("reset repository: %v", err)
	}
	if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}

	return nil
}
