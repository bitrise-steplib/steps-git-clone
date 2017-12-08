package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

const (
	retryCount = 2
	waitTime   = 5 // seconds
)

// Git ...
var Git git.Git

func printLogAndExportEnv(format, env string) error {
	l, err := runForOutput(Git.Log(format))
	if err != nil {
		return err
	}

	log.Printf("=> %s\n   value: %s\n", env, l)
	if err := exportEnvironmentWithEnvman(env, l); err != nil {
		return fmt.Errorf("envman export failed, error: %v", err)
	}
	return nil
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := command.New("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

func mainE() error {
	config, errs := newConfig()
	if len(errs) > 0 {
		text := ""
		for _, err := range errs {
			text += err.Error() + "\n"
		}
		return fmt.Errorf("Invalid inputs:\n%s", text)
	}
	config.print()
	Git, err := git.New(config.CloneIntoDir)
	if err != nil {
		return fmt.Errorf("Can't create Git project, error: %v", err)
	}

	checkoutArg := getCheckoutArg(config.Commit, config.Tag, config.Branch)

	originPresent, err := isOriginPresent(config.CloneIntoDir, config.RepositoryURL)
	if err != nil {
		return fmt.Errorf("Can't check if origin is presented, error: %v", err)
	}

	if originPresent && config.ResetRepository {
		if err := resetRepo(); err != nil {
			return fmt.Errorf("Can't reset repository, error: %v", err)
		}
	}

	if err := run(Git.Init()); err != nil {
		return fmt.Errorf("Can't init repository, error: %v", err)
	}

	if !originPresent {
		if err := run(Git.RemoteAdd("origin", config.RepositoryURL)); err != nil {
			return fmt.Errorf("Can't add remote repository (%s), error: %v", config.RepositoryURL, err)
		}
	}

	if isPR(config.PRRepositoryCloneURL, config.PRMergeBranch, config.PRID) {
		if !config.ManualMerge || isPrivate(config.PRRepositoryCloneURL) && isFork(config.RepositoryURL, config.PRRepositoryCloneURL) {
			if err := autoMerge(config.PRMergeBranch, config.BranchDest, config.BuildURL,
				config.BuildAPIToken, config.CloneDepth, config.PRID); err != nil {
				return fmt.Errorf("Failed, error: %v", err)
			}
		} else {
			if err := manualMerge(config.RepositoryURL, config.PRRepositoryCloneURL, config.Branch,
				config.Commit, config.BranchDest, config.CloneDepth); err != nil {
				return fmt.Errorf("Failed, error: %v", err)
			}
		}
	} else if checkoutArg != "" {
		if err := checkout(checkoutArg, config.CloneDepth); err != nil {
			return fmt.Errorf("Failed, error: %v", err)
		}
	}

	if config.UpdateSubmodules {
		if err := run(Git.SubmoduleUpdate()); err != nil {
			return fmt.Errorf("Submodule update failed, error: %v", err)
		}
	}

	if checkoutArg != "" {
		log.Infof("\nExporting git logs\n")

		for format, env := range map[string]string{
			`"%H"`:  "GIT_CLONE_COMMIT_HASH",
			`"%s"`:  "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
			`"%b"`:  "GIT_CLONE_COMMIT_MESSAGE_BODY",
			`"%an"`: "GIT_CLONE_COMMIT_AUTHOR_NAME",
			`"%ae"`: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
			`"%cn"`: "GIT_CLONE_COMMIT_COMMITER_NAME",
			`"%ce"`: "GIT_CLONE_COMMIT_COMMITER_EMAIL",
		} {
			if err := printLogAndExportEnv(format, env); err != nil {
				return fmt.Errorf("Git log failed, error: %v", err)
			}
		}

		count, err := runForOutput(Git.RevList("HEAD", "--count"))
		if err != nil {
			return fmt.Errorf("Git rev-list command failed, error: %v", err)
		}

		log.Printf("=> %s\n   value: %s\n", "GIT_CLONE_COMMIT_COUNT", count)
		if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COUNT", count); err != nil {
			return fmt.Errorf("Envman export failed, error: %v", err)
		}
	}

	return nil
}

func main() {
	if err := mainE(); err != nil {
		log.Errorf("ERROR: %+v", err)
		os.Exit(1)
	}
	log.Donef("\nSuccess")
}
