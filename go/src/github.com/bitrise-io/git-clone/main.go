package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bitrise-io/git-clone/gitutil"
	log "github.com/bitrise-io/git-clone/logger"
	"github.com/bitrise-io/git-clone/retry"
)

const (
	retryCount = 2
	waitTime   = 20 //seconds
)

// -----------------------
// --- Functions
// -----------------------

func validateRequiredInput(key, value string) {
	if value == "" {
		log.Fail("Missing required input: %s", key)
	}
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	envman := exec.Command("envman", "add", "--key", keyStr)
	envman.Stdin = strings.NewReader(valueStr)
	envman.Stdout = os.Stdout
	envman.Stderr = os.Stderr
	return envman.Run()
}

// -----------------------
// --- Main
// -----------------------
func main() {
	//
	// Validate options
	repositoryURL := os.Getenv("repository_url")
	cloneIntoDir := os.Getenv("clone_into_dir")
	commit := os.Getenv("commit")
	tag := os.Getenv("tag")
	branch := os.Getenv("branch")
	pullRequestID := os.Getenv("pull_request_id")
	cloneDepth := os.Getenv("clone_depth")

	log.Configs(repositoryURL, commit, tag, branch, pullRequestID, cloneIntoDir, cloneDepth)

	validateRequiredInput("repository_url", repositoryURL)
	validateRequiredInput("clone_into_dir", cloneIntoDir)

	// git
	log.Info("Git clone repository")

	git, err := gitutil.NewHelper(cloneIntoDir, repositoryURL)
	if err != nil {
		log.Fail("Failed to create git helper, error: %s", err)
	}

	git.ConfigureCheckoutParam(pullRequestID, commit, tag, branch, cloneDepth)

	if err := git.Init(); err != nil {
		log.Fail("Failed, error: %s", err)
	}

	if err := git.RemoteAdd(); err != nil {
		log.Fail("Failed, error: %s", err)
	}

	if err := retry.Times(retryCount).Wait(waitTime).Retry(func(attempt uint) error {
		if attempt > 0 {
			log.Warn("Retrying...")
		}

		fetchErr := git.Fetch()
		if fetchErr != nil {
			log.Warn("%d attempt failed:", attempt)
			fmt.Println(fetchErr.Error())
		}

		return fetchErr
	}); err != nil {
		log.Fail("Failed, error: %s", err)
	}

	if git.ShouldCheckout() {
		if err := git.Checkout(); err != nil {
			if !git.ShouldTryFetchUnshallow() {
				log.Fail("Failed, error: %s", err)
			}

			log.Warn("Failed, error: %s", err)
			log.Warn("Unshallow...")

			if err := retry.Times(retryCount).Wait(waitTime).Retry(func(attempt uint) error {
				if attempt > 0 {
					log.Warn("Retrying...")
				}

				fetchShallowErr := git.FetchUnshallow()
				if fetchShallowErr != nil {
					log.Warn("%d attempt failed:", attempt)
					fmt.Println(fetchShallowErr.Error())
				}

				return fetchShallowErr
			}); err != nil {
				log.Fail("Failed, error: %s", err)
			}

			if err := git.Checkout(); err != nil {
				log.Fail("Failed, error: %s", err)
			}
		}

		if err := retry.Times(retryCount).Wait(waitTime).Retry(func(attempt uint) error {
			if attempt > 0 {
				log.Warn("Retrying...")
			}

			submoduleErr := git.SubmoduleUpdate()
			if submoduleErr != nil {
				log.Warn("%d attempt failed:", attempt)
				fmt.Println(submoduleErr.Error())
			}

			return submoduleErr
		}); err != nil {
			log.Fail("Failed, error: %s", err)
		}

		log.Info("Exporting git logs")
		if commitHash, err := git.LogCommitHash(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_HASH", commitHash)
		}

		if commitMessageSubject, err := git.LogCommitMessageSubject(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_SUBJECT", commitMessageSubject)
		}

		if commitMessageBody, err := git.LogCommitMessageBody(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_BODY", commitMessageBody)
		}

		if commitAuthorName, err := git.LogAuthorName(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_NAME", commitAuthorName)
		}

		if commitAuthorEmail, err := git.LogAuthorEmail(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_EMAIL", commitAuthorEmail)
		}

		if commitCommiterName, err := git.LogCommiterName(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_NAME", commitCommiterName)
		}

		if commitCommiterEmail, err := git.LogCommiterEmail(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_EMAIL", commitCommiterEmail)
		}
	}
}
