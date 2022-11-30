package gitclone

import (
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/v2/analytics"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

type stepTracker struct {
	tracker analytics.Tracker
	logger  log.Logger
}

func newStepTracker(envRepo env.Repository, logger log.Logger) stepTracker {
	p := analytics.Properties{
		"build_slug":  envRepo.Get("BITRISE_BUILD_SLUG"),
		"is_pr_build": envRepo.Get("PR") == "true",
	}
	return stepTracker{
		tracker: analytics.NewDefaultTracker(logger, p),
		logger:  logger,
	}
}

func (t *stepTracker) logCheckout(duration time.Duration, method CheckoutMethod, remoteURL string) {
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
	t.tracker.Enqueue("step_git_clone_checkout", p)
}

func (t *stepTracker) logSubmoduleUpdate(duration time.Duration) {
	p := analytics.Properties{
		"duration_s": duration.Truncate(time.Second).Seconds(),
	}
	t.tracker.Enqueue("step_git_clone_submodule_updated", p)
}

func (t *stepTracker) wait() {
	t.tracker.Wait()
}
