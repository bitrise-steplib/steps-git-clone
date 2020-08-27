package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/envman/envman"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

type config struct {
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
	AutoMerge        bool   `env:"auto_merge,opt[yes,no]"`
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

func mainE() error {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Error: %s\n", err)
	}
	stepconf.Print(cfg)

	maxEnvLength, err := getMaxEnvLength()
	if err != nil {
		failf("failed to set commit message length: %s\n", err)
	}

	gitCmd, err := git.New(cfg.CloneIntoDir)
	if err != nil {
		return fmt.Errorf("create gitCmd project, error: %v", err)
	}
	checkoutArg := getCheckoutArg(cfg.Commit, cfg.Tag, cfg.Branch)

	originPresent, err := isOriginPresent(gitCmd, cfg.CloneIntoDir, cfg.RepositoryURL)
	if err != nil {
		return fmt.Errorf("check if origin is presented, error: %v", err)
	}

	if originPresent && cfg.ResetRepository {
		if err := resetRepo(gitCmd); err != nil {
			return fmt.Errorf("reset repository, error: %v", err)
		}
	}
	if err := run(gitCmd.Init()); err != nil {
		return fmt.Errorf("init repository, error: %v", err)
	}
	if !originPresent {
		if err := run(gitCmd.RemoteAdd("origin", cfg.RepositoryURL)); err != nil {
			return fmt.Errorf("add remote repository (%s), error: %v", cfg.RepositoryURL, err)
		}
	}

	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if isPR {
		if !cfg.ManualMerge || isPrivate(cfg.PRRepositoryURL) && isFork(cfg.RepositoryURL, cfg.PRRepositoryURL) {
			if err := autoMerge(gitCmd, cfg.PRMergeBranch, cfg.BranchDest, cfg.BuildURL,
				cfg.BuildAPIToken, cfg.CloneDepth, cfg.PRID); err != nil {
				return fmt.Errorf("auto merge, error: %v", err)
			}
		} else {
			if err := manualMerge(gitCmd, cfg.RepositoryURL, cfg.PRRepositoryURL, cfg.Branch,
				cfg.Commit, cfg.BranchDest, cfg.AutoMerge); err != nil {
				return fmt.Errorf("manual merge, error: %v", err)
			}
		}
	} else if checkoutArg != "" {
		
		if err := checkout(gitCmd, checkoutArg, cfg.Branch, cfg.CloneDepth, cfg.Tag != ""); err != nil {
			return fmt.Errorf("checkout (%s): %v", checkoutArg, err)
		}
		// Update branch: 'git fetch' followed by a 'git merge' is the same as 'git pull'.
		log.Printf("ZZZ cfg.AutoMerge")
		if checkoutArg == cfg.Branch && cfg.AutoMerge{
			if err := run(gitCmd.Merge("origin/" + cfg.Branch)); err != nil {
				return fmt.Errorf("merge %q: %v", cfg.Branch, err)
			}
		}
	}

	if cfg.UpdateSubmodules {
		if err := run(gitCmd.SubmoduleUpdate()); err != nil {
			return fmt.Errorf("submodule update: %v", err)
		}
	}

	if isPR {
		if err := run(gitCmd.Checkout("--detach")); err != nil {
			return fmt.Errorf("detach head: %v", err)
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
				return fmt.Errorf("gitCmd log failed, error: %v", err)
			}
		}

		count, err := output(gitCmd.RevList("HEAD", "--count"))
		if err != nil {
			return fmt.Errorf("get rev-list, error: %v", err)
		}

		log.Printf("=> %s\n   value: %s\n", "GIT_CLONE_COMMIT_COUNT", count)
		if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COUNT", count); err != nil {
			return fmt.Errorf("envman export, error: %v", err)
		}
	}

	return nil
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	if err := mainE(); err != nil {
		failf("ERROR: %v", err)
	}
	log.Donef("\nSuccess")
}
