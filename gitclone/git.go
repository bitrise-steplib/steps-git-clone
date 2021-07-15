package gitclone

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
)

const (
	checkoutFailedTag = "checkout_failed"
	fetchFailedTag    = "fetch_failed"
	jobsFlag          = "--jobs=10"
)

var runner CommandRunner = DefaultRunner{}

func isOriginPresent(gitCmd git.Git, dir, repoURL string) (bool, error) {
	absDir, err := pathutil.AbsPath(dir)
	if err != nil {
		return false, err
	}

	gitDir := filepath.Join(absDir, ".git")
	if exist, err := pathutil.IsDirExists(gitDir); err != nil {
		return false, err
	} else if exist {
		remotes, err := runner.RunForOutput(gitCmd.RemoteList())
		if err != nil {
			return false, err
		}

		if !strings.Contains(remotes, repoURL) {
			return false, fmt.Errorf(".git folder exists in the directory (%s), but using a different remote", dir)
		}
		return true, nil
	}

	return false, nil
}

func resetRepo(gitCmd git.Git) error {
	if err := runner.Run(gitCmd.Reset("--hard", "HEAD")); err != nil {
		return err
	}
	if err := runner.Run(gitCmd.Clean("-x", "-d", "-f")); err != nil {
		return err
	}
	if err := runner.Run(gitCmd.SubmoduleForeach(gitCmd.Reset("--hard", "HEAD"))); err != nil {
		return err
	}
	return runner.Run(gitCmd.SubmoduleForeach(gitCmd.Clean("-x", "-d", "-f")))
}

func isFork(repoURL, prRepoURL string) bool {
	return prRepoURL != "" && getRepo(repoURL) != getRepo(prRepoURL)
}

// formats:
// https://hostname/owner/repository.git
// git@hostname:owner/repository.git
// ssh://git@hostname:port/owner/repository.git
func getRepo(url string) string {
	var host, repo string
	switch {
	case strings.HasPrefix(url, "https://"):
		url = strings.TrimPrefix(url, "https://")
		idx := strings.Index(url, "/")
		host, repo = url[:idx], url[idx+1:]
	case strings.HasPrefix(url, "git@"):
		url = url[strings.Index(url, "@")+1:]
		idx := strings.Index(url, ":")
		host, repo = url[:idx], url[idx+1:]
	case strings.HasPrefix(url, "ssh://"):
		url = url[strings.Index(url, "@")+1:]
		if strings.Contains(url, ":") {
			idxColon, idxSlash := strings.Index(url, ":"), strings.Index(url, "/")
			host, repo = url[:idxColon], url[idxSlash+1:]
		} else {
			idx := strings.Index(url, "/")
			host, repo = url[:idx], url[idx+1:]
		}
	}
	return host + "/" + strings.TrimSuffix(repo, ".git")
}

func isPrivate(repoURL string) bool {
	return strings.HasPrefix(repoURL, "git")
}

// If incoming branch matches to pull/x/merge pattern headBranchRefs
// converts it to remote ref (pull/x/head) and local ref (pull/x)
// If it does not match it keeps original name.
func headBranchRefs(mergeBranch string) (remoteRef, localRef string) {
	var re = regexp.MustCompile("^pull/(.*)/merge$")
	if re.MatchString(mergeBranch) {
		branchID := re.ReplaceAllString(mergeBranch, "$1")
		remoteRef = fmt.Sprintf("refs/pull/%s/head", branchID)
		localRef = fmt.Sprintf("pull/%s", branchID)
		return
	}

	remoteRef = "refs/heads/" + mergeBranch
	localRef = mergeBranch
	return
}

type getAvailableBranches func() (map[string][]string, error)

func listBranches(gitCmd git.Git) getAvailableBranches {
	return func() (map[string][]string, error) {
		if err := runner.Run(gitCmd.Fetch(jobsFlag)); err != nil {
			return nil, err
		}
		out, err := runner.RunForOutput(gitCmd.Branch("-r"))
		if err != nil {
			return nil, err
		}

		return parseListBranchesOutput(out), nil
	}
}

func parseListBranchesOutput(output string) map[string][]string {
	lines := strings.Split(output, "\n")
	branchesByRemote := map[string][]string{}
	for _, line := range lines {
		line = strings.Trim(line, " ")
		split := strings.Split(line, "/")

		remote := split[0]
		branch := ""
		if len(split) > 1 {
			branch = strings.Join(split[1:], "/")
			branches := branchesByRemote[remote]
			branches = append(branches, branch)
			branchesByRemote[remote] = branches
		}
	}
	return branchesByRemote
}

func handleCheckoutError(callback getAvailableBranches, tag string, err error, shortMsg string, branch string) error {
	// We were checking out a branch (not tag or commit)
	if branch != "" {
		branchesByRemote, branchesErr := callback()
		branches := branchesByRemote[originRemoteName]
		// There was no error grabbing the available branches
		// And the current branch is not present in the list
		if branchesErr == nil && !sliceutil.IsStringInSlice(branch, branches) {
			return newStepErrorWithBranchRecommendations(
				tag,
				err,
				shortMsg,
				branch,
				branches,
			)
		}
	}

	return newStepError(
		tag,
		err,
		shortMsg,
	)
}
