package gitclone

import (
	"fmt"

	"github.com/bitrise-io/envman/envman"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
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
	ManualMerge      bool   `env:"manual_merge,opt[yes,no]"`
}

const (
	trimEnding              = "..."
	originRemoteName        = "origin"
	forkRemoteName          = "fork"
	updateSubmodelFailedTag = "update_submodule_failed"
	sparseCheckoutFailedTag = "sparse_checkout_failed"
)

type commitInfo struct {
	envKey string
	cmd    *command.Model
}

func exportCommitInfo(gitCmd git.Git, gitRef string, isPR bool, maxEnvLength int) error {
	commitInfos := []commitInfo{
		{
			envKey: "GIT_CLONE_COMMIT_AUTHOR_NAME",
			cmd:    gitCmd.Log(`%an`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
			cmd:    gitCmd.Log(`%ae`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_HASH",
			cmd:    gitCmd.Log(`%H`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
			cmd:    gitCmd.Log(`%s`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_MESSAGE_BODY",
			cmd:    gitCmd.Log(`%b`, gitRef),
		},
	}
	nonPROnlyInfos := []commitInfo{
		{
			envKey: "GIT_CLONE_COMMIT_COMMITER_NAME",
			cmd:    gitCmd.Log(`%cn`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_COMMITER_EMAIL",
			cmd:    gitCmd.Log(`%ce`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_COUNT",
			cmd:    gitCmd.RevList("HEAD", "--count"),
		},
	}

	if !isPR {
		commitInfos = append(commitInfos, nonPROnlyInfos...)
	} else {
		log.Printf("Git commiter name/email and commit count is not exported for Pull Requests.")
	}

	for _, commitInfo := range commitInfos {
		if err := printLogAndExportEnv(commitInfo.cmd, commitInfo.envKey, maxEnvLength); err != nil {
			return err
		}
	}

	return nil
}

func printLogAndExportEnv(command *command.Model, env string, maxEnvLength int) error {
	l, err := runner.RunForOutput(command)
	if err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	if (env == "GIT_CLONE_COMMIT_MESSAGE_SUBJECT" || env == "GIT_CLONE_COMMIT_MESSAGE_BODY") && len(l) > maxEnvLength {
		tv := l[:maxEnvLength-len(trimEnding)] + trimEnding
		log.Printf("Value %s  is bigger than maximum env variable size, trimming", env)
		l = tv
	}

	log.Printf("=> %s\n   value: %s", env, l)
	if err := tools.ExportEnvironmentWithEnvman(env, l); err != nil {
		return fmt.Errorf("envman export failed: %v", err)
	}
	return nil
}

func getMaxEnvLength() (int, error) {
	configs, err := envman.GetConfigs()
	if err != nil {
		return 0, err
	}

	return configs.EnvBytesLimitInKB * 1024, nil
}

func checkoutState(gitCmd git.Git, cfg Config, patch patchSource) (strategy checkoutStrategy, isPR bool, err error) {
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
		log.Infof("Checkout strategy used: %T", checkoutStrategy)
		return nil, false, err
	}

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
	maxEnvLength, err := getMaxEnvLength()
	if err != nil {
		return newStepError(
			"get_max_commit_msg_length_failed",
			fmt.Errorf("failed to set commit message length: %s", err),
			"Getting allowed commit message length failed",
		)
	}

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
		if err := updateSubmodules(gitCmd, cfg); err != nil {
			return err
		}
	}

	if ref := checkoutStrategy.getBuildTriggerRef(); ref != "" {
		fmt.Println()
		log.Infof("Exporting commit details")
		if err := exportCommitInfo(gitCmd, ref, isPR, maxEnvLength); err != nil {
			return newStepError("export_envs_failed", err, "Exporting envs failed")
		}
	} else {
		fmt.Println()
		log.Warnf(`Can not export commit information like commit message and author as it is not available.
This may happen when using Bitbucket with the "Manual merge" input set to 'yes' (using a Diff file).`)
	}

	return nil
}
