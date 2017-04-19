package gitutil

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// ---------------------
//	Model
// ---------------------

// PullRequestHelper ...
type PullRequestHelper struct {
	pullRequestID            string
	pullRequestRepositoryURI string
	pullRequestRemoteName    string
	pullRequestBranch        string
	pullRequestMergeBranch   string
	pullRequestDiffPath      string
}

// Helper ...
type Helper struct {
	destinationDir string
	remoteURI      string
	remoteName     string

	checkoutParam     string
	checkoutTag       bool
	pullRequestHelper PullRequestHelper
	cloneDepth        string
	originPresent     bool
}

// NewHelper ...
func NewHelper(destinationDir, remoteURI string, resetRepository bool) (Helper, error) {
	if destinationDir == "" {
		return Helper{}, errors.New("destination dir path is empty")
	}

	if remoteURI == "" {
		return Helper{}, errors.New("remote URI is empty")
	}

	// Expand destination dir
	fullDestinationDir, err := pathutil.AbsPath(destinationDir)
	if err != nil {
		return Helper{}, err
	}

	helper := Helper{
		destinationDir: fullDestinationDir,
		remoteURI:      remoteURI,
		remoteName:     "origin",
	}

	// Check if .git exist
	gitDirPth := filepath.Join(fullDestinationDir, ".git")
	if exist, err := pathutil.IsDirExists(gitDirPth); err != nil {
		return Helper{}, err
	} else if exist {
		remotes, err := helper.RemoteList()
		if err != nil {
			return Helper{}, err
		}

		if !strings.Contains(remotes, remoteURI) {
			return Helper{}, fmt.Errorf(".git folder already exists in the destination dir: %s, using a different remote", fullDestinationDir)
		}

		if resetRepository {
			if err = helper.Clean(); err != nil {
				return Helper{}, err
			}
		}
		helper.originPresent = true
	}

	// Create destination dir if not exist
	if exist, err := pathutil.IsDirExists(fullDestinationDir); err != nil {
		return Helper{}, err
	} else if !exist {
		if err := os.MkdirAll(fullDestinationDir, 0777); err != nil {
			return Helper{}, err
		}
	}

	return helper, nil
}

// ConfigureCheckout ...
func (helper *Helper) ConfigureCheckout(pullRequestID, pullRequestURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, buildURL, buildAPIToken string) {
	if pullRequestID != "" && pullRequestMergeBranch != "" {
		helper.ConfigureCheckoutWithPullRequestID(pullRequestID, pullRequestMergeBranch, cloneDepth)
	} else {
		if pullRequestID != "" && pullRequestURI != "" && branchDest != "" {
			helper.ConfigureCheckoutWithPullRequestURI(pullRequestID, helper.remoteURI, branchDest, cloneDepth)

			// try to get diff file
			diffPath, err := helper.savePullRequestDiff(buildURL, buildAPIToken)
			if err == nil {
				// if we are able to get the diff file,
				// we should checkout the destination branch
				helper.ConfigureCheckoutWithParams("", "", branchDest, cloneDepth)

				if exists, err := pathutil.IsPathExists(diffPath); err == nil && exists {
					if diffContent, err := fileutil.ReadStringFromFile(diffPath); err == nil && diffContent != "" {
						helper.pullRequestHelper.pullRequestDiffPath = diffPath
					}
				}
			} else {
				// if not diff file is available, we should
				// checkout the PR's commit hash
				helper.ConfigureCheckoutWithParams(commitHash, tag, branch, cloneDepth)
				helper.remoteURI = pullRequestURI
			}
		} else {
			helper.ConfigureCheckoutWithParams(commitHash, tag, branch, cloneDepth)
		}
	}
}

// ConfigureCheckoutWithPullRequestURI ...
func (helper *Helper) ConfigureCheckoutWithPullRequestURI(pullRequestID, pullRequestURI, pullRequestBranch, cloneDepth string) {
	helper.pullRequestHelper = PullRequestHelper{
		pullRequestID:            pullRequestID,
		pullRequestRepositoryURI: pullRequestURI,
		pullRequestBranch:        pullRequestBranch,
	}

	helper.cloneDepth = cloneDepth
}

// ConfigureCheckoutWithPullRequestID ...
func (helper *Helper) ConfigureCheckoutWithPullRequestID(pullRequestID, pullRequestMergeBranch, cloneDepth string) {
	helper.checkoutParam = "pull/" + pullRequestID
	helper.pullRequestHelper = PullRequestHelper{
		pullRequestID:          pullRequestID,
		pullRequestMergeBranch: pullRequestMergeBranch,
	}

	helper.cloneDepth = cloneDepth
}

// ConfigureCheckoutWithParams ...
func (helper *Helper) ConfigureCheckoutWithParams(commitHash, tag, branch, cloneDepth string) {
	if commitHash != "" {
		helper.checkoutParam = commitHash
	} else if tag != "" {
		helper.checkoutParam = tag
		helper.checkoutTag = true
	} else if branch != "" {
		helper.checkoutParam = branch
	}

	helper.cloneDepth = cloneDepth
}

func runCommandInDirWithEnvsAndOutput(cmdSlice []string, dir string, envs []string) (string, string, error) {
	cmd, err := command.NewFromSlice(cmdSlice)
	if err != nil {
		return "", "", err
	}

	if len(envs) > 0 {
		cmd.SetEnvs(envs...)
	}

	if dir != "" {
		cmd.SetDir(dir)
	}

	log.Printf("=> %s", command.PrintableCommandArgs(false, cmdSlice))

	var errBuffer bytes.Buffer
	errWriter := bufio.NewWriter(&errBuffer)
	cmd.SetStderr(errWriter)

	var outBuffer bytes.Buffer
	outWriter := bufio.NewWriter(&outBuffer)
	cmd.SetStdout(outWriter)

	if err := cmd.Run(); err != nil {
		if errorutil.IsExitStatusError(err) {
			if !errorutil.IsExitStatusErrorStr(errBuffer.String()) {
				return "", "", errors.New(errBuffer.String())
			}

			if !errorutil.IsExitStatusErrorStr(outBuffer.String()) {
				return "", "", errors.New(outBuffer.String())
			}
		}

		return "", "", err
	}

	return outBuffer.String(), errBuffer.String(), nil
}

func runCommandInDirWithEnvs(cmdSlice []string, dir string, envs []string) error {
	_, _, err := runCommandInDirWithEnvsAndOutput(cmdSlice, dir, envs)
	return err
}

func runCommandInDir(cmdSlice []string, dir string) error {
	return runCommandInDirWithEnvs(cmdSlice, dir, []string{})
}

// IsOriginPresented ...
func (helper Helper) IsOriginPresented() bool {
	return helper.originPresent
}

// Init ...
func (helper Helper) Init() error {
	cmdSlice := createGitCmdSlice("init")

	return runCommandInDir(cmdSlice, helper.destinationDir)
}

// RemoteList ...
func (helper Helper) RemoteList() (string, error) {
	cmdSlice := createGitCmdSlice("remote", "-v")

	remotes, _, err := runCommandInDirWithEnvsAndOutput(cmdSlice, helper.destinationDir, []string{})
	return remotes, err
}

// RemoteAdd ...
func (helper Helper) RemoteAdd() error {
	return helper.RemoteAddWithParams(helper.remoteName, helper.remoteURI)
}

// RemoteAddWithParams ...
func (helper Helper) RemoteAddWithParams(remoteName, remoteURI string) error {
	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass("remote", "add", remoteName, remoteURI)

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

// RemoteRemove ...
func (helper Helper) RemoteRemove(remoteName string) error {
	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass("remote", "rm", remoteName)

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

// Clean ...
func (helper Helper) Clean() error {
	cmdSlice := createGitCmdSlice("reset", "--hard", "HEAD")
	if err := runCommandInDir(cmdSlice, helper.destinationDir); err != nil {
		return err
	}

	cmdSlice = createGitCmdSlice("clean", "-xdf")
	if err := runCommandInDir(cmdSlice, helper.destinationDir); err != nil {
		return err
	}

	cmdSlice = createGitCmdSlice("submodule", "foreach", "git", "reset", "--hard", "HEAD")
	if err := runCommandInDir(cmdSlice, helper.destinationDir); err != nil {
		return err
	}

	cmdSlice = createGitCmdSlice("submodule", "foreach", "git", "clean", "-xdf")
	if err := runCommandInDir(cmdSlice, helper.destinationDir); err != nil {
		return err
	}

	return nil
}

// Fetch ...
func (helper Helper) Fetch() error {
	params := []string{"fetch"}
	if helper.pullRequestHelper.pullRequestID != "" && helper.pullRequestHelper.pullRequestMergeBranch != "" {
		params = append(params, helper.remoteName, helper.pullRequestHelper.pullRequestMergeBranch+":"+helper.checkoutParam)
	}
	if helper.cloneDepth != "" {
		params = append(params, "--depth="+helper.cloneDepth)
	}

	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass(params...)

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

// FetchTags ...
func (helper Helper) FetchTags() error {
	params := []string{"fetch", "--tags"}
	if helper.pullRequestHelper.pullRequestID != "" && helper.pullRequestHelper.pullRequestBranch != "" {
		params = append(params, helper.remoteName, helper.pullRequestHelper.pullRequestBranch+":"+helper.checkoutParam)
	}
	if helper.cloneDepth != "" {
		params = append(params, "--depth="+helper.cloneDepth)
	}

	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass(params...)

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

// ShouldCheckout ...
func (helper Helper) ShouldCheckout() bool {
	return (helper.checkoutParam != "")
}

// ShouldCheckoutTag ...
func (helper Helper) ShouldCheckoutTag() bool {
	return helper.checkoutTag
}

// Checkout ...
func (helper Helper) Checkout() error {
	cmdSlice := createGitCmdSlice("checkout", helper.checkoutParam)

	return runCommandInDir(cmdSlice, helper.destinationDir)
}

// ShouldTryFetchUnshallow ...
func (helper Helper) ShouldTryFetchUnshallow() bool {
	return (helper.cloneDepth != "")
}

// FetchUnshallow ...
func (helper Helper) FetchUnshallow() error {
	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass("fetch", "--unshallow")

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

// ShouldMergePullRequest ...
func (helper Helper) ShouldMergePullRequest() bool {
	return (helper.pullRequestHelper.pullRequestRepositoryURI != "" && helper.pullRequestHelper.pullRequestBranch != "")
}

// savePullRequestDiff ...
func (helper Helper) savePullRequestDiff(buildURL, buildAPIToken string) (string, error) {
	uri := fmt.Sprintf("%s/diff.txt?api_token=%s", buildURL, buildAPIToken)
	response, err := http.Get(uri)
	if err != nil {
		return "", err
	}

	if response.StatusCode != 200 {
		return "", errors.New("Diff is not available")
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Errorf("Failed to close response body, error: %s", err)
		}
	}()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	diffFileName := fmt.Sprintf("%s.diff", helper.pullRequestHelper.pullRequestID)
	diffFile, err := ioutil.TempFile("", diffFileName)
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

// MergePullRequest ...
func (helper Helper) MergePullRequest() error {
	// Applying diff if available
	if helper.pullRequestHelper.pullRequestDiffPath != "" {
		cmdSlice := createGitCmdSlice("apply", helper.pullRequestHelper.pullRequestDiffPath)
		if err := runCommandInDir(cmdSlice, helper.destinationDir); err != nil {
			return err
		}

		return nil
	}

	// Normal merge
	if helper.remoteURI == helper.pullRequestHelper.pullRequestRepositoryURI {
		helper.pullRequestHelper.pullRequestRemoteName = helper.remoteName
	} else {
		helper.pullRequestHelper.pullRequestRemoteName = "upstream"

		remotes, err := helper.RemoteList()
		if err != nil {
			return err
		}

		if strings.Contains(remotes, helper.pullRequestHelper.pullRequestRemoteName+"\t") {
			if err := helper.RemoteRemove(helper.pullRequestHelper.pullRequestRemoteName); err != nil {
				return err
			}
		}
		if err := helper.RemoteAddWithParams(helper.pullRequestHelper.pullRequestRemoteName, helper.pullRequestHelper.pullRequestRepositoryURI); err != nil {
			return err
		}
	}

	cmdSlice := createGitCmdSlice("fetch", helper.pullRequestHelper.pullRequestRemoteName, helper.pullRequestHelper.pullRequestBranch)
	if err := runCommandInDir(cmdSlice, helper.destinationDir); err != nil {
		return err
	}

	cmdSlice = createGitCmdSlice("merge", helper.pullRequestHelper.pullRequestRemoteName+"/"+helper.pullRequestHelper.pullRequestBranch)
	if err := runCommandInDir(cmdSlice, helper.destinationDir); err != nil {
		return err
	}

	return nil
}

// SubmoduleUpdate ...
func (helper Helper) SubmoduleUpdate() error {
	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass("submodule", "update", "--init", "--recursive")

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

func runLogCommand(cmdSlice []string, dir string) (string, error) {
	out, err := command.New(cmdSlice[0], cmdSlice[1:]...).SetDir(dir).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.Trim(out, `"`), nil
}

// LogCommitHash ...
func (helper Helper) LogCommitHash() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%H"`)
	return runLogCommand(cmdSlice, helper.destinationDir)
}

// LogCommitMessageSubject ...
func (helper Helper) LogCommitMessageSubject() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%s"`)
	return runLogCommand(cmdSlice, helper.destinationDir)
}

// LogCommitMessageBody ...
func (helper Helper) LogCommitMessageBody() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%b"`)
	return runLogCommand(cmdSlice, helper.destinationDir)
}

// LogAuthorName ...
func (helper Helper) LogAuthorName() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%an"`)
	return runLogCommand(cmdSlice, helper.destinationDir)
}

// LogAuthorEmail ...
func (helper Helper) LogAuthorEmail() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%ae"`)
	return runLogCommand(cmdSlice, helper.destinationDir)
}

// LogCommiterName ...
func (helper Helper) LogCommiterName() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%cn"`)
	return runLogCommand(cmdSlice, helper.destinationDir)
}

// LogCommiterEmail ...
func (helper Helper) LogCommiterEmail() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%ce"`)
	return runLogCommand(cmdSlice, helper.destinationDir)
}

// ---------------------
//	Methods
// ---------------------

func createGitCmdSlice(params ...string) []string {
	return append([]string{"git"}, params...)
}

func createGitCmdSliceWithGitDontAskpass(params ...string) ([]string, []string) {
	return createGitCmdSlice(params...), []string{"GIT_ASKPASS=echo"}
}

func createGitLogCmdSlice(params ...string) []string {
	return append([]string{"git", "log", "-1"}, params...)
}

func properReturn(err error, out string) error {
	if err == nil {
		return nil
	}

	if errorutil.IsExitStatusError(err) && out != "" {
		return errors.New(out)
	}
	return err
}
