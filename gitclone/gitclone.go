package gitclone

import (
	"fmt"
	"time"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

// Config is the git clone step configuration
type Config struct {
	RepositoryURL string `env:"repository_url,required"`
	CloneIntoDir  string `env:"clone_into_dir,required"`
	Commit        string `env:"commit"`
	Tag           string `env:"tag"`
	Branch        string `env:"branch"`

	PRDestBranch          string `env:"branch_dest"`
	PRID                  int    `env:"pull_request_id"`
	PRSourceRepositoryURL string `env:"pull_request_repository_url"`
	PRMergeBranch         string `env:"pull_request_merge_branch"`
	PRHeadBranch          string `env:"pull_request_head_branch"`

	ResetRepository      bool     `env:"reset_repository,opt[Yes,No]"`
	CloneDepth           int      `env:"clone_depth"`
	FetchTags            bool     `env:"fetch_tags,opt[yes,no]"`
	SubmoduleUpdateDepth int      `env:"submodule_update_depth"`
	ShouldMergePR        bool     `env:"merge_pr,opt[yes,no]"`
	SparseDirectories    []string `env:"sparse_directories,multiline"`

	BuildURL         string `env:"build_url"`
	BuildAPIToken    string `env:"build_api_token"`
	UpdateSubmodules bool   `env:"update_submodules,opt[yes,no]"`
}

const (
	trimEnding              = "..."
	originRemoteName        = "origin"
	forkRemoteName          = "fork"
	updateSubmodelFailedTag = "update_submodule_failed"
	sparseCheckoutFailedTag = "sparse_checkout_failed"
)

var logger = log.NewLogger()
var tracker = newStepTracker(env.NewRepository(), logger)

func checkoutState(gitCmd git.Git, cfg Config, patch patchSource) (strategy checkoutStrategy, isPR bool, err error) {
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
		logger.Infof("Checkout strategy used: %T", checkoutStrategy)
		return nil, false, err
	}

	checkoutDuration := time.Since(checkoutStartTime).Round(time.Second)
	logger.Println()
	logger.Infof("Fetch and checkout took %s", checkoutDuration)
	tracker.logCheckout(checkoutDuration, checkoutMethod, cfg.RepositoryURL)

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
			updateSubmodelFailedTag,
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

// Execute is the entry point of the git clone process
func Execute(cfg Config) error {
	defer tracker.wait()

	cmdFactory := command.NewFactory(env.NewRepository())

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
			"Checking wether origin is present failed",
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

	if err := setupSparseCheckout(gitCmd, cfg.SparseDirectories); err != nil {
		return err
	}

	checkoutStrategy, isPR, err := checkoutState(gitCmd, cfg, defaultPatchSource{})
	if err != nil {
		return err
	}

	if cfg.UpdateSubmodules {
		startTime := time.Now()
		if err := updateSubmodules(gitCmd, cfg); err != nil {
			return err
		}
		updateTime := time.Since(startTime).Round(time.Second)
		logger.Println()
		logger.Infof("Updating submodules took %s", updateTime)
		tracker.logSubmoduleUpdate(updateTime)
	}

	fmt.Println()
	logger.Infof("Exporting commit details")
	ref := checkoutStrategy.getBuildTriggerRef()
	if ref == "" {
		logger.Warnf(`Can't export commit information (commit message and author) as it is not available.
This is a limitation of Bitbucket webhooks when the PR source repo (a fork) is not accessible.
Try using the env vars based on the webhook contents instead, such as $BITRISE_GIT_COMMIT and $BITRISE_GIT_MESSAGE`)
		return nil
	}

	exporter := newOutputExporter(cmdFactory, gitCmd)
	if err := exporter.exportCommitInfo(ref, isPR); err != nil {
		return newStepError("export_envs_failed", err, "Exporting envs failed")
	}

	return nil
}
