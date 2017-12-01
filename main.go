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

// vars ...
var (
	configs     ConfigsModel
	Git         *git.Git
	checkoutArg string
)

func initConfig() error {
	configs = createConfigsModelFromEnvs()
	fmt.Println()
	configs.print()
	if err := configs.validate(); err != nil {
		return fmt.Errorf("issue with input: %v", err)
	}
	fmt.Println()
	Git = git.New(configs.CloneIntoDir)
	checkoutArg = setCheckoutArg()
	return nil
}

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func printLog(format, env string) error {
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

func main() {
	if err := initConfig(); err != nil {
		fail("Failed, error: %v", err)
	}

	originPresent, err := isOriginPresent(configs.CloneIntoDir, configs.RepositoryURL)
	if err != nil {
		fail("Can't check if origin is presented, error: %v", err)
	}

	if originPresent && configs.ResetRepository == "yes" {
		if err := resetRepo(); err != nil {
			fail("Can't reset repository, error: %v", err)
		}
	}

	if err := os.MkdirAll(configs.CloneIntoDir, 0755); err != nil {
		fail("Can't create directory (%s), error: %v", configs.CloneIntoDir, err)
	}

	if err := run(Git.Init()); err != nil {
		fail("Can't init repository, error: %v", err)
	}

	if !originPresent {
		if err := run(Git.RemoteAdd("origin", configs.RepositoryURL)); err != nil {
			fail("Can't add remote repository (%s), error: %v", configs.RepositoryURL, err)
		}
	}

	if err := runWithRetry(func() *command.Model {
		if configs.CloneDepth != "" {
			return Git.Fetch("--depth=" + configs.CloneDepth)
		}
		return Git.Fetch()
	}); err != nil {
		fail("Fetch failed, error: %v", err)
	}

	if isPR() {
		if configs.ManualMerge == "yes" {
			if err := manualMerge(true); err != nil {
				fail("Failed, error: %v", err)
			}
		} else {
			if err := autoMerge(true); err != nil {
				fail("Failed, error: %v", err)
			}
		}
	} else if checkoutArg != "" {
		if err := run(Git.Checkout(checkoutArg)); err != nil {
			if configs.CloneDepth == "" {
				fail("Checkout failed (%s), error: %v", checkoutArg, err)
			}
			log.Warnf("Checkout failed, error: %v\nUnshallow...", err)

			if err := runWithRetry(func() *command.Model {
				return Git.Fetch("--unshallow")
			}); err != nil {
				fail("Fetch failed, error: %v", err)
			}
			if err := run(Git.Checkout(checkoutArg)); err != nil {
				fail("Checkout failed (%s), error: %v", checkoutArg, err)
			}
		}
	}

	if configs.UpdateSubmodules == "yes" {
		if err := run(Git.SubmoduleUpdate()); err != nil {
			fail("Submodule update failed, error: %v", err)
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
			if err := printLog(format, env); err != nil {
				fail("Git log failed, error: %v", err)
			}
		}

		count, err := runForOutput(Git.RevList("HEAD", "--count"))
		if err != nil {
			fail("Git rev-list command failed, error: %v", err)
		}

		log.Printf("=> %s\n   value: %s\n", "GIT_CLONE_COMMIT_COUNT", count)
		if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COUNT", count); err != nil {
			fail("Envman export failed, error: %v", err)
		}
	}

	log.Donef("Success")
}
