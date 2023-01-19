package gitclone

import (
	"fmt"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/go-utils/command/git"
)

const (
	trimEnding               = "..."
	originRemoteName         = "origin"
	forkRemoteName           = "fork"
	updateSubmoduleFailedTag = "update_submodule_failed"
	sparseCheckoutFailedTag  = "sparse_checkout_failed"
)

// Config is the git clone step configuration
type Config struct {
	ShouldMergePR bool

	CloneIntoDir         string
	CloneDepth           int
	UpdateSubmodules     bool
	SubmoduleUpdateDepth int
	FetchTags            bool
	SparseDirectories    []string

	RepositoryURL         string
	Commit                string
	Tag                   string
	Branch                string
	PRDestBranch          string
	PRSourceRepositoryURL string
	PRMergeBranch         string
	PRHeadBranch          string

	ResetRepository bool
	BuildURL        string
	BuildAPIToken   string
}

type GitCloner struct {
	logger     log.Logger
	tracker    StepTracker
	cmdFactory command.Factory
}

func NewGitCloner(logger log.Logger, tracker StepTracker, cmdFactory command.Factory) GitCloner {
	return GitCloner{
		logger:     logger,
		tracker:    tracker,
		cmdFactory: cmdFactory,
	}
}

// CheckoutState is the entry point of the git clone process
func (g GitCloner) CheckoutState(cfg Config) error {
	defer g.tracker.wait()

	gitCmd, err := git.New(cfg.CloneIntoDir)
	if err != nil {
		return newStepError(
			"git_new",
			fmt.Errorf("failed to create git project directory: %v", err),
			"Creating new git project directory failed",
		)
	}

	originPresent, err := isOriginPresent(gitCmd, cfg.CloneIntoDir, cfg.RepositoryURL)
	if err != nil {
		return newStepError(
			"check_origin_present_failed",
			fmt.Errorf("checking if origin is present failed: %v", err),
			"Checking whether origin is present failed",
		)
	}

	if originPresent && cfg.ResetRepository {
		if err := resetRepo(gitCmd); err != nil {
			return newStepError(
				"reset_repository_failed",
				fmt.Errorf("reset repository failed: %v", err),
				"Resetting repository failed",
			)
		}
	}
	if err := runner.Run(gitCmd.Init()); err != nil {
		return newStepError(
			"init_git_failed",
			fmt.Errorf("initializing repository failed: %v", err),
			"Initializing git has failed",
		)
	}
	if !originPresent {
		if err := runner.Run(gitCmd.RemoteAdd(originRemoteName, cfg.RepositoryURL)); err != nil {
			return newStepError(
				"add_remote_failed",
				fmt.Errorf("adding remote repository failed (%s): %v", cfg.RepositoryURL, err),
				"Adding remote repository failed",
			)
		}
	}

	// Disable automatic GC as it may be triggered by other git commands (making run times nondeterministic).
	// And we run in ephemeral VMs anyway, so GC isn't really needed.
	// https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-gc.html
	err = runner.Run(gitCmd.Config("gc.auto", "0"))
	if err != nil {
		return newStepError(
			"disable_gc",
			fmt.Errorf("failed to disable GC: %v", err),
			"Failed to disable git garbage collection",
		)
	}

	if err := setupSparseCheckout(gitCmd, cfg.SparseDirectories); err != nil {
		return err
	}

	checkoutStrategy, isPR, err := g.checkoutState(gitCmd, cfg, defaultPatchSource{})
	if err != nil {
		return err
	}

	if cfg.UpdateSubmodules {
		startTime := time.Now()
		if err := updateSubmodules(gitCmd, cfg); err != nil {
			return err
		}
		updateTime := time.Since(startTime).Round(time.Second)
		g.logger.Println()
		g.logger.Infof("Updating submodules took %s", updateTime)
		g.tracker.logSubmoduleUpdate(updateTime)
	}

	fmt.Println()
	g.logger.Infof("Exporting commit details")
	ref := checkoutStrategy.getBuildTriggerRef()
	if ref == "" {
		g.logger.Warnf(`Can't export commit information (commit message and author) as it is not available.
This is a limitation of Bitbucket webhooks when the PR source repo (a fork) is not accessible.
Try using the env vars based on the webhook contents instead, such as $BITRISE_GIT_COMMIT and $BITRISE_GIT_MESSAGE`)
		return nil
	}

	exporter := newOutputExporter(g.logger, g.cmdFactory, gitCmd)
	if err := exporter.exportCommitInfo(ref, isPR); err != nil {
		return newStepError("export_envs_failed", err, "Exporting envs failed")
	}

	return nil
}

func (g GitCloner) checkoutState(gitCmd git.Git, cfg Config, patch patchSource) (strategy checkoutStrategy, isPR bool, err error) {
	checkoutStartTime := time.Now()
	checkoutMethod, diffFile := selectCheckoutMethod(cfg, patch)

	fetchOpts := selectFetchOptions(checkoutMethod, cfg.CloneDepth, cfg.FetchTags, cfg.UpdateSubmodules, len(cfg.SparseDirectories) != 0)

	checkoutStrategy, err := createCheckoutStrategy(checkoutMethod, cfg, diffFile)
	if err != nil {
		return nil, false, err
	}
	if checkoutStrategy == nil {
		return nil, false, fmt.Errorf("failed to select a checkout stategy")
	}

	if err := checkoutStrategy.do(gitCmd, fetchOpts, selectFallbacks(checkoutMethod, fetchOpts)); err != nil {
		g.logger.Infof("Checkout strategy used: %T", checkoutStrategy)
		return nil, false, err
	}

	checkoutDuration := time.Since(checkoutStartTime).Round(time.Second)
	g.logger.Println()
	g.logger.Infof("Fetch and checkout took %s", checkoutDuration)
	g.tracker.logCheckout(checkoutDuration, checkoutMethod, cfg.RepositoryURL)

	return checkoutStrategy, isPRCheckout(checkoutMethod), nil
}

func updateSubmodules(gitCmd git.Git, cfg Config) error {
	var opts []string
	opts = append(opts, jobsFlag)

	if cfg.SubmoduleUpdateDepth > 0 {
		opts = append(opts, fmt.Sprintf("--depth=%d", cfg.SubmoduleUpdateDepth))
	}

	if err := runner.Run(gitCmd.SubmoduleUpdate(opts...)); err != nil {
		return newStepError(
			updateSubmoduleFailedTag,
			fmt.Errorf("submodule update: %v", err),
			"Updating submodules has failed",
		)
	}

	return nil
}

func setupSparseCheckout(gitCmd git.Git, sparseDirectories []string) error {
	if len(sparseDirectories) == 0 {
		return nil
	}

	initCommand := gitCmd.SparseCheckoutInit(true)
	if err := runner.Run(initCommand); err != nil {
		return newStepError(
			sparseCheckoutFailedTag,
			fmt.Errorf("initializing sparse-checkout config failed: %v", err),
			"Initializing sparse-checkout config has failed",
		)
	}

	sparseSetCommand := gitCmd.SparseCheckoutSet(sparseDirectories...)
	if err := runner.Run(sparseSetCommand); err != nil {
		return newStepError(
			sparseCheckoutFailedTag,
			fmt.Errorf("updating sparse-checkout config failed: %v", err),
			"Updating sparse-checkout config has failed",
		)
	}

	// Enable partial clone support for the remote
	sparseConfigCmd := gitCmd.Config("extensions.partialClone", originRemoteName, "--local")
	if err := runner.Run(sparseConfigCmd); err != nil {
		return newStepError(
			sparseCheckoutFailedTag,
			fmt.Errorf("enable partial clone support for the remote has failed: %v", err),
			"Enable partial clone support for the remote has failed",
		)
	}

	return nil
}
