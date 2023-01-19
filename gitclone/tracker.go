package gitclone

import (
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/v2/analytics"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

type StepTracker struct {
	tracker analytics.Tracker
	logger  log.Logger
}

func NewStepTracker(envRepo env.Repository, logger log.Logger) StepTracker {
	p := analytics.Properties{
		"build_slug":  envRepo.Get("BITRISE_BUILD_SLUG"),
		"is_pr_build": envRepo.Get("PR") == "true",
	}
	return StepTracker{
		tracker: analytics.NewDefaultTracker(logger, p),
		logger:  logger,
	}
}

func (t *StepTracker) logCheckout(duration time.Duration, method CheckoutMethod, remoteURL string) {
	var remoteType = "other"
	if strings.Contains(remoteURL, "github.com") {
		remoteType = "github.com"
	} else if strings.Contains(remoteURL, "gitlab.com") {
		remoteType = "gitlab.com"
	} else if strings.Contains(remoteURL, "bitbucket.org") {
		remoteType = "bitbucket.org"
	}

	p := analytics.Properties{
		"duration_s":  duration.Truncate(time.Second).Seconds(),
		"method":      method.String(),
		"remote_type": remoteType,
	}
	t.tracker.Enqueue("step_git_clone_fetch_and_checkout", p)
}

func (t *StepTracker) logSubmoduleUpdate(duration time.Duration) {
	p := analytics.Properties{
		"duration_s": duration.Truncate(time.Second).Seconds(),
	}
	t.tracker.Enqueue("step_git_clone_submodule_updated", p)
}

func (t *StepTracker) wait() {
	t.tracker.Wait()
}
