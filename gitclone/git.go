package gitclone

import (
	"fmt"
	"path/filepath"
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

var runner CommandRunner = &DefaultRunner{}

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

type getAvailableBranches func() (map[string][]string, error)

func listBranches(gitCmd git.Git) getAvailableBranches {
	return func() (map[string][]string, error) {
		remoteList, err := runner.RunForOutput(gitCmd.RemoteList())
		if err != nil {
			return nil, err
		}

		remoteBranches, err := runner.RunForOutput(gitCmd.RemoteBranches())
		if err != nil {
			return nil, err
		}

		return parseListBranchesOutput(remoteList, remoteBranches), nil
	}
}

func parseListBranchesOutput(remotes, branches string) map[string][]string {
	parsedRemotes := extractRemotes(remotes)
	branchesByRemote := extractBranches(parsedRemotes, branches)

	return branchesByRemote
}

// extractRemotes parses the output of `git remote -v` and returns a map of remote URLs to their names
// Example of what such an output looks like:
//
//	origin  git_url1 (fetch)
//	origin  git_url1 (push)
//	upstream  git_url2 (fetch)
//	upstream  git_url2 (push)
//
// (Name of the remote, \t, URL, space, (fetch|push), \n)
func extractRemotes(remotes string) map[string]string {
	parsedRemotes := make(map[string]string)

	for _, line := range strings.Split(remotes, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		split := strings.Split(line, "\t")
		if len(split) < 2 {
			continue
		}

		name := split[0]

		split = strings.Split(split[1], " ")
		url := split[0]

		parsedRemotes[url] = name
	}

	return parsedRemotes
}

// extractBranches parses the output of `git ls-remote -b` and returns a map of remote names to their branches
// Example of what such an output looks like:
//
//	From git_url
//	a50bdc5182e3e7c292f7c8c881a1a0d9476c8eda        refs/heads/A
//	25c9d97e5c9e7f4c9ad25597f8e1265af07869d7        refs/heads/B
//	5b3dfe10c52cf5c762740ea5cfa619d82d70e205        refs/heads/C
//	f6cdf8f2132f22516bb4d71e326e66a45850eb04        refs/heads/D
//	b30b826ac9594330d77554f30103492409035aee        refs/heads/master
//
// (From, space, URL, \n)
// (SHA, \t, refs/heads/, branch name, \n)
func extractBranches(remotes map[string]string, branches string) map[string][]string {
	branchesByRemote := make(map[string][]string)
	currentRemote := ""
	for _, line := range strings.Split(branches, "\n") {
		if strings.HasPrefix(line, "From ") {
			repoURL := strings.TrimPrefix(line, "From ")
			currentRemote = remotes[repoURL]

			continue
		}

		split := strings.Split(line, "\t")
		if len(split) < 2 || currentRemote == "" {
			continue
		}

		branch := strings.TrimPrefix(split[1], refsHeadsPrefix)
		branches := branchesByRemote[currentRemote]
		branches = append(branches, branch)
		branchesByRemote[currentRemote] = branches
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

func isWorkingTreeClean(gitCmd git.Git) (bool, error) {
	// Despite the flag name, `--porcelain` is the plumbing format to use in scripts:
	// https://git-scm.com/docs/git-status#Documentation/git-status.txt---porcelainltversiongt
	out, err := gitCmd.Status("--porcelain").RunAndReturnTrimmedOutput()
	if err != nil {
		return false, fmt.Errorf("git status check: %s", err)
	}
	return strings.TrimSpace(out) == "", nil
}
