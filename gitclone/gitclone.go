package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/envman/envman"
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

	BranchDest      string `env:"branch_dest"`
	PRID            int    `env:"pull_request_id"`
	PRRepositoryURL string `env:"pull_request_repository_url"`
	PRMergeBranch   string `env:"pull_request_merge_branch"`
	ResetRepository bool   `env:"reset_repository,opt[Yes,No]"`
	CloneDepth      int    `env:"clone_depth"`

	BuildURL         string `env:"build_url"`
	BuildAPIToken    string `env:"build_api_token"`
	UpdateSubmodules bool   `env:"update_submodules,opt[yes,no]"`
	ManualMerge      bool   `env:"manual_merge,opt[yes,no]"`
}

const (
	trimEnding = "..."
)

func printLogAndExportEnv(gitCmd git.Git, format, env string, maxEnvLength int) error {
	l, err := output(gitCmd.Log(format))
	if err != nil {
		return err
	}

	if (env == "GIT_CLONE_COMMIT_MESSAGE_SUBJECT" || env == "GIT_CLONE_COMMIT_MESSAGE_BODY") && len(l) > maxEnvLength {
		tv := l[:maxEnvLength-len(trimEnding)] + trimEnding
		log.Printf("Value %s  is bigger than maximum env variable size, trimming", env)
		l = tv
	}

	log.Printf("=> %s\n   value: %s\n", env, l)
	if err := exportEnvironmentWithEnvman(env, l); err != nil {
		return fmt.Errorf("envman export, error: %v", err)
	}
	return nil
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := command.New("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

func getMaxEnvLength() (int, error) {
	configs, err := envman.GetConfigs()
	if err != nil {
		return 0, err
	}

	return configs.EnvBytesLimitInKB * 1024, nil
}

// Execute is the entry point of the git clone process
func Execute(cfg Config) error {
	maxEnvLength, err := getMaxEnvLength()
	if err != nil {
		return &StepError{
			Tag:      "get_max_commit_msg_length_failed",
			ShortMsg: "Getting allowed commit message length failed",
			Err:      fmt.Errorf("failed to set commit message length: %s", err),
		}
	}

	gitCmd, err := git.New(cfg.CloneIntoDir)
	if err != nil {
		return &StepError{
			Tag:      "git_new",
			ShortMsg: "Creating new git project directory failed",
			Err:      fmt.Errorf("failed to create git project directory: %v", err),
		}
	}
	checkoutArg := getCheckoutArg(cfg.Commit, cfg.Tag, cfg.Branch)

	originPresent, err := isOriginPresent(gitCmd, cfg.CloneIntoDir, cfg.RepositoryURL)
	if err != nil {
		return &StepError{
			Tag:      "check_origin_present_failed",
			ShortMsg: "Checking wether origin is present failed",
			Err:      fmt.Errorf("checking if origin is present failed: %v", err),
		}
	}

	if originPresent && cfg.ResetRepository {
		if err := resetRepo(gitCmd); err != nil {
			return &StepError{
				Tag:      "reset_repository_failed",
				ShortMsg: "Resetting repository failed",
				Err:      fmt.Errorf("reset repository failed: %v", err),
			}
		}
	}
	if err := run(gitCmd.Init()); err != nil {
		return &StepError{
			Tag:      "init_git_failed",
			ShortMsg: "Initializing git has failed",
			Err:      fmt.Errorf("initializing repository failed: %v", err),
		}
	}
	if !originPresent {
		if err := run(gitCmd.RemoteAdd("origin", cfg.RepositoryURL)); err != nil {
			return &StepError{
				Tag:      "add_remote_failed",
				ShortMsg: "Adding remote repository failed",
				Err:      fmt.Errorf("adding remote repository failed (%s): %v", cfg.RepositoryURL, err),
			}
		}
	}

	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if isPR {
		if !cfg.ManualMerge || isPrivate(cfg.PRRepositoryURL) && isFork(cfg.RepositoryURL, cfg.PRRepositoryURL) {
			if err := autoMerge(gitCmd, cfg.PRMergeBranch, cfg.BranchDest, cfg.BuildURL,
				cfg.BuildAPIToken, cfg.CloneDepth, cfg.PRID); err != nil {
				return &StepError{
					Tag:      "auto_merge_failed",
					ShortMsg: "Merging pull request failed",
					Err:      fmt.Errorf("merging PR (automatic) failed: %v", err),
				}
			}
		} else {
			if err := manualMerge(gitCmd, cfg.RepositoryURL, cfg.PRRepositoryURL, cfg.Branch,
				cfg.Commit, cfg.BranchDest); err != nil {
				return &StepError{
					Tag:      "manual_merge_failed",
					ShortMsg: "Merging pull request failed",
					Err:      fmt.Errorf("merging PR (manual) failed: %v", err),
				}
			}
		}
	} else if checkoutArg != "" {
		if err := checkout(gitCmd, checkoutArg, cfg.Branch, cfg.CloneDepth, cfg.Tag != ""); err != nil {
			return err
		}
		// Update branch: 'git fetch' followed by a 'git merge' is the same as 'git pull'.
		if checkoutArg == cfg.Branch {
			if err := run(gitCmd.Merge("origin/" + cfg.Branch)); err != nil {
				return &StepError{
					Tag:      "update_branch_failed",
					ShortMsg: "Updating branch failed",
					Err:      fmt.Errorf("updating branch (merge) failed %q: %v", cfg.Branch, err),
				}
			}
		}
	}

	if cfg.UpdateSubmodules {
		if err := run(gitCmd.SubmoduleUpdate()); err != nil {
			return &StepError{
				Tag:      "update_submodule_failed",
				ShortMsg: "Updating submodules has failed",
				Err:      fmt.Errorf("submodule update: %v", err),
			}
		}
	}

	if isPR {
		if err := run(gitCmd.Checkout("--detach")); err != nil {
			return &StepError{
				Tag:      "detach_head_failed",
				ShortMsg: "Detaching head failed",
				Err:      fmt.Errorf("detach head failed: %v", err),
			}
		}
	}

	if checkoutArg != "" {
		log.Infof("\nExporting git logs\n")

		for format, env := range map[string]string{
			`%H`:  "GIT_CLONE_COMMIT_HASH",
			`%s`:  "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
			`%b`:  "GIT_CLONE_COMMIT_MESSAGE_BODY",
			`%an`: "GIT_CLONE_COMMIT_AUTHOR_NAME",
			`%ae`: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
			`%cn`: "GIT_CLONE_COMMIT_COMMITER_NAME",
			`%ce`: "GIT_CLONE_COMMIT_COMMITER_EMAIL",
		} {
			if err := printLogAndExportEnv(gitCmd, format, env, maxEnvLength); err != nil {
				return &StepError{
					Tag:      "export_envs_failed",
					ShortMsg: "Exporting envs failed",
					Err:      fmt.Errorf("gitCmd log failed: %v", err),
				}
			}
		}

		count, err := output(gitCmd.RevList("HEAD", "--count"))
		if err != nil {
			return &StepError{
				Tag:      "count_commits_failed",
				ShortMsg: "Counting commits failed",
				Err:      fmt.Errorf("get rev-list failed: %v", err),
			}
		}

		log.Printf("=> %s\n   value: %s\n", "GIT_CLONE_COMMIT_COUNT", count)
		if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COUNT", count); err != nil {
			return &StepError{
				Tag:      "export_envs_commit_count_failed",
				ShortMsg: "Exporting commit count env failed",
				Err:      fmt.Errorf("envman export failed: %v", err),
			}
		}
	}

	return nil
}
