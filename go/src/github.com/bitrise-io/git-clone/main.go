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
	waitTime   = 5 //seconds
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
	resetRepository := os.Getenv("reset_repository") == "Yes"

	log.Configs(repositoryURL, cloneIntoDir, commit, tag, branch, pullRequestID, cloneDepth)

	validateRequiredInput("repository_url", repositoryURL)
	validateRequiredInput("clone_into_dir", cloneIntoDir)

	// git
	log.Info("Git clone repository")

	git, err := gitutil.NewHelper(cloneIntoDir, repositoryURL, resetRepository)
	if err != nil {
		log.Fail("Failed to create git helper, error: %s", err)
	}

	git.ConfigureCheckoutParam(pullRequestID, commit, tag, branch, cloneDepth)

	if err := git.Init(); err != nil {
		log.Fail("Failed, error: %s", err)
	}

	if !git.IsOriginPresented() {
		if err := git.RemoteAdd(); err != nil {
			log.Fail("Failed, error: %s", err)
		}
	}

	if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
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
		if git.ShouldCheckoutTag() {
			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warn("Retrying...")
				}

				fetchErr := git.FetchTags()
				if fetchErr != nil {
					log.Warn("%d attempt failed:", attempt)
					fmt.Println(fetchErr.Error())
				}

				return fetchErr
			}); err != nil {
				log.Fail("Failed, error: %s", err)
			}
		}

		if err := git.Checkout(); err != nil {
			if !git.ShouldTryFetchUnshallow() {
				log.Fail("Failed, error: %s", err)
			}

			log.Warn("Failed, error: %s", err)
			log.Warn("Unshallow...")

			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
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

		if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
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
			log.Details("=> GIT_CLONE_COMMIT_HASH")
			log.Details("   value: %s", commitHash)
			fmt.Println()

			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_HASH", commitHash)
		}

		if commitMessageSubject, err := git.LogCommitMessageSubject(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			log.Details("=> GIT_CLONE_COMMIT_MESSAGE_SUBJECT")
			log.Details("   value: %s", commitMessageSubject)
			fmt.Println()

			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_SUBJECT", commitMessageSubject)
		}

		if commitMessageBody, err := git.LogCommitMessageBody(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			log.Details("=> GIT_CLONE_COMMIT_MESSAGE_BODY")
			log.Details("   value: %s", commitMessageBody)
			fmt.Println()

			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_BODY", commitMessageBody)
		}

		if commitAuthorName, err := git.LogAuthorName(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			log.Details("=> GIT_CLONE_COMMIT_AUTHOR_NAME")
			log.Details("   value: %s", commitAuthorName)
			fmt.Println()

			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_NAME", commitAuthorName)
		}

		if commitAuthorEmail, err := git.LogAuthorEmail(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			log.Details("=> GIT_CLONE_COMMIT_AUTHOR_EMAIL")
			log.Details("   value: %s", commitAuthorEmail)
			fmt.Println()

			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_EMAIL", commitAuthorEmail)
		}

		if commitCommiterName, err := git.LogCommiterName(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			log.Details("=> GIT_CLONE_COMMIT_COMMITER_NAME")
			log.Details("   value: %s", commitCommiterName)
			fmt.Println()

			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_NAME", commitCommiterName)
		}

		if commitCommiterEmail, err := git.LogCommiterEmail(); err != nil {
			log.Fail("Git log failed, error: %s", err)
		} else {
			log.Details("=> GIT_CLONE_COMMIT_COMMITER_EMAIL")
			log.Details("   value: %s", commitCommiterEmail)
			fmt.Println()

			exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_EMAIL", commitCommiterEmail)
		}
	}

	log.Done("Success")
}
