package logger

import (
	"fmt"
	"os"
)

// Fail ...
func Fail(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", errorMsg)
	os.Exit(1)
}

// Warn ...
func Warn(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	fmt.Printf("\x1b[33;1m%s\x1b[0m\n", errorMsg)
}

// Info ...
func Info(format string, v ...interface{}) {
	fmt.Println()
	errorMsg := fmt.Sprintf(format, v...)
	fmt.Printf("\x1b[34;1m%s\x1b[0m\n", errorMsg)
}

// Details ...
func Details(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	fmt.Printf("  %s\n", errorMsg)
}

// Done ...
func Done(format string, v ...interface{}) {
	errorMsg := fmt.Sprintf(format, v...)
	fmt.Printf("  \x1b[32;1m%s\x1b[0m\n", errorMsg)
}

// Configs ...
func Configs(repositoryURL, cloneIntoDir, commit, tag, branch, branchDest, pullRequestURI, pullRequestBranch, pullRequestID, buildURL, buildAPIToken, cloneDepth string, resetRepository bool) {
	Info("Configs:")

	Details("repository_url: %s", repositoryURL)
	Details("clone_into_dir: %s", cloneIntoDir)
	Details("commit: %s", commit)
	Details("tag: %s", tag)
	Details("branch: %s", branch)
	Details("branch_dest: %s", branchDest)
	Details("pull_request_repository_url: %s", pullRequestURI)
	Details("pull_request_merge_branch: %s", pullRequestBranch)
	Details("pull_request_id: %s", pullRequestID)
	Details("clone_depth: %s", cloneDepth)
	Details("reset_repository: %t", resetRepository)
	Details("build_url: %s", buildURL)
	Details("build_api_token: %s", buildAPIToken)
}
