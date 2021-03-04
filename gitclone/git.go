package gitclone

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
)

const (
	checkoutFailedTag = "checkout_failed"
	fetchFailedTag    = "fetch_failed"
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

func getCheckoutArg(commit, tag, branch string) string {
	switch {
	case commit != "":
		return commit
	case tag != "":
		return tag
	case branch != "":
		return branch
	default:
		return ""
	}
}

func getDiffFile(buildURL, apiToken string, prID int) (string, error) {
	url := fmt.Sprintf("%s/diff.txt?api_token=%s", buildURL, apiToken)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body, error: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Can't download diff file, HTTP status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	diffFile, err := ioutil.TempFile("", fmt.Sprintf("%d.diff", prID))
	if err != nil {
		return "", err
	}

	if _, err := diffFile.Write(body); err != nil {
		return "", err
	}
	if err := diffFile.Close(); err != nil {
		return "", err
	}

	return diffFile.Name(), nil
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

// If incoming branch matches to pull/x/merge pattern fetchArg
// converts it to pull/x/head:pull/x otherwise original name is kept.
func fetchArg(mergeBranch string) string {
	var re = regexp.MustCompile("^pull/(.*)/merge$")
	if re.MatchString(mergeBranch) {
		return re.ReplaceAllString(mergeBranch, "refs/pull/$1/head:pull/$1")
	}
	return "refs/heads/" + mergeBranch + ":" + mergeBranch
}

func mergeArg(mergeBranch string) string {
	return strings.TrimSuffix(mergeBranch, "/merge")
}

func autoMerge(gitCmd git.Git, mergeBranch, branchDest, buildURL, apiToken string, depth, id int) error {
	var opts []string
	if depth != 0 {
		opts = append(opts, "--depth="+strconv.Itoa(depth))
	}
	opts = append(opts, defaultRemoteName, "refs/heads/"+branchDest)

	if err := runner.RunWithRetry(gitCmd.Fetch(opts...)); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if mergeBranch != "" {
		if err := runner.RunWithRetry(gitCmd.Fetch(defaultRemoteName, fetchArg(mergeBranch))); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v",
				mergeBranch, err)
		}
		if err := pull(gitCmd, branchDest); err != nil {
			return fmt.Errorf("pull failed (%s), error: %v", branchDest, err)
		}
		if err := runner.Run(gitCmd.Merge(mergeArg(mergeBranch))); err != nil {
			if depth == 0 {
				return fmt.Errorf("merge %q: %v", mergeArg(mergeBranch), err)
			}
			log.Warnf("Merge failed, error: %v\nReset repository, then unshallow...", err)
			if err := resetRepo(gitCmd); err != nil {
				return fmt.Errorf("reset repository, error: %v", err)
			}
			if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
				return fmt.Errorf("fetch failed, error: %v", err)
			}
			if err := runner.Run(gitCmd.Merge(mergeArg(mergeBranch))); err != nil {
				return fmt.Errorf("merge %q: %v", mergeArg(mergeBranch), err)
			}
		}
	} else if patch, err := getDiffFile(buildURL, apiToken, id); err == nil {
		if err := runner.Run(gitCmd.Checkout(branchDest)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", branchDest, err)
		}
		if err := runner.Run(gitCmd.Apply(patch)); err != nil {
			return fmt.Errorf("can't apply patch (%s), error: %v", patch, err)
		}
	} else {
		return fmt.Errorf("there is no Pull Request branch and can't download diff file")
	}
	return nil
}

func manualMerge(gitCmd git.Git, repoURL, prRepoURL, branch, commit, branchDest string) error {
	if err := runner.RunWithRetry(gitCmd.Fetch(defaultRemoteName, "refs/heads/"+branchDest)); err != nil {
		return fmt.Errorf("fetch failed, error: %v", err)
	}
	if err := pull(gitCmd, branchDest); err != nil {
		return fmt.Errorf("pull failed (%s), error: %v", branchDest, err)
	}
	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	if isFork(repoURL, prRepoURL) {
		if err := runner.Run(gitCmd.RemoteAdd("fork", prRepoURL)); err != nil {
			return fmt.Errorf("couldn't add remote (%s), error: %v", prRepoURL, err)
		}
		if err := runner.RunWithRetry(gitCmd.Fetch("fork", "refs/heads/"+branch)); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v", branch, err)
		}
		if err := runner.Run(gitCmd.Merge("fork/" + branch)); err != nil {
			return fmt.Errorf("merge failed (fork/%s), error: %v", branch, err)
		}
	} else {
		if err := runner.Run(gitCmd.Fetch(defaultRemoteName, "refs/heads/"+branch)); err != nil {
			return fmt.Errorf("fetch failed, error: %v", err)
		}
		if err := runner.Run(gitCmd.Merge(commit)); err != nil {
			return fmt.Errorf("merge failed (%s), error: %v", commit, err)
		}
	}

	return nil
}

type getAvailableBranches func() (map[string][]string, error)

func listBranches(gitCmd git.Git) getAvailableBranches {
	return func() (map[string][]string, error) {
		if err := runner.Run(gitCmd.Fetch()); err != nil {
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
		branches := branchesByRemote[defaultRemoteName]
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

func checkout(gitCmd git.Git, arg, branch string, depth int, isTag bool) error {
	var opts []string
	if depth != 0 {
		opts = append(opts, "--depth="+strconv.Itoa(depth))
	}
	if isTag {
		opts = append(opts, "--tags")
	}
	if branch == arg || (branch != "" && isTag) {
		opts = append(opts, defaultRemoteName, "refs/heads/"+branch)
	}

	if err := runner.RunWithRetry(gitCmd.Fetch(opts...)); err != nil {
		return handleCheckoutError(
			listBranches(gitCmd),
			fetchFailedTag,
			fmt.Errorf("fetch failed, error: %v", err),
			"Fetching repository has failed",
			branch,
		)
	}

	if err := runner.Run(gitCmd.Checkout(arg)); err != nil {
		if depth == 0 {
			return handleCheckoutError(
				listBranches(gitCmd),
				checkoutFailedTag,
				fmt.Errorf("checkout failed (%s), error: %v", arg, err),
				"Checkout has failed",
				branch,
			)
		}
		log.Warnf("Checkout failed, error: %v\nUnshallow...", err)
		if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
			return newStepError(
				"fetch_unshallow_failed",
				fmt.Errorf("fetch (unshallow) failed, error: %v", err),
				"Fetching with unshallow parameter has failed",
			)
		}
		if err := runner.Run(gitCmd.Checkout(arg)); err != nil {
			return newStepError(
				"checkout_unshallow_failed",
				fmt.Errorf("checkout failed (%s), error: %v", arg, err),
				"Checkout after unshallow fetch has failed",
			)
		}
	}

	return nil
}

// pull is a 'git fetch' followed by a 'git merge' which is the same as 'git pull'.
func pull(gitCmd git.Git, branchDest string) error {
	if err := runner.Run(gitCmd.Checkout(branchDest)); err != nil {
		return fmt.Errorf("checkout failed (%s), error: %v", branchDest, err)
	}
	return runner.Run(gitCmd.Merge("origin/" + branchDest))
}
