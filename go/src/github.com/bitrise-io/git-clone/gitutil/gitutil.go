package gitutil

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/bitrise-io/git-clone/logger"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

// ---------------------
//	Model
// ---------------------

// Helper ...
type Helper struct {
	destinationDir string
	remoteURI      string

	checkoutParam string
	checkoutTag   bool
	pullRequestID string
	cloneDepth    string
}

// NewHelper ...
func NewHelper(destinationDir, remoteURI string) (Helper, error) {
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

	// Check if .git exist
	gitDirPth := filepath.Join(fullDestinationDir, ".git")
	if exist, err := pathutil.IsDirExists(gitDirPth); err != nil {
		return Helper{}, err
	} else if exist {
		return Helper{}, fmt.Errorf(".git folder already exists in the destination dir: %s", fullDestinationDir)
	}

	// Create destination dir if not exist
	if exist, err := pathutil.IsDirExists(fullDestinationDir); err != nil {
		return Helper{}, err
	} else if !exist {
		if err := os.MkdirAll(fullDestinationDir, 0777); err != nil {
			return Helper{}, err
		}
	}

	return Helper{
		destinationDir: fullDestinationDir,
		remoteURI:      remoteURI,
	}, nil
}

// ConfigureCheckoutParam ...
func (helper *Helper) ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth string) {
	if pullRequestID != "" {
		helper.checkoutParam = "pull/" + pullRequestID
		helper.pullRequestID = pullRequestID
	} else if commitHash != "" {
		helper.checkoutParam = commitHash
	} else if tag != "" {
		helper.checkoutParam = tag
		helper.checkoutTag = true
	} else if branch != "" {
		helper.checkoutParam = branch
	}

	helper.cloneDepth = cloneDepth
}

func runCommandInDirWithEnvs(cmdSlice []string, dir string, envs []string) error {
	cmd, err := cmdex.NewCommandFromSlice(cmdSlice)
	if err != nil {
		return err
	}

	if len(envs) > 0 {
		cmd.SetEnvs(envs)
	}

	if dir != "" {
		cmd.SetDir(dir)
	}

	log.Details("=> %s", cmdex.PrintableCommandArgs(false, cmdSlice))

	var errBuffer bytes.Buffer
	errWriter := bufio.NewWriter(&errBuffer)
	cmd.SetStderr(errWriter)

	var outBuffer bytes.Buffer
	outWriter := bufio.NewWriter(&outBuffer)
	cmd.SetStdout(outWriter)

	if err := cmd.Run(); err != nil {
		if errorutil.IsExitStatusError(err) {
			if !errorutil.IsExitStatusErrorStr(errBuffer.String()) {
				return errors.New(errBuffer.String())
			}

			if !errorutil.IsExitStatusErrorStr(outBuffer.String()) {
				return errors.New(outBuffer.String())
			}
		}

		return err
	}

	return nil
}

func runCommandInDir(cmdSlice []string, dir string) error {
	return runCommandInDirWithEnvs(cmdSlice, dir, []string{})
}

// Init ...
func (helper Helper) Init() error {
	cmdSlice := createGitCmdSlice("init")

	return runCommandInDir(cmdSlice, helper.destinationDir)
}

// RemoteAdd ...
func (helper Helper) RemoteAdd() error {
	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass("remote", "add", "origin", helper.remoteURI)

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

// Fetch ...
func (helper Helper) Fetch() error {
	params := []string{"fetch"}
	if helper.pullRequestID != "" {
		params = append(params, "origin", "pull/"+helper.pullRequestID+"/merge:"+helper.checkoutParam)
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
	if helper.pullRequestID != "" {
		params = append(params, "origin", "pull/"+helper.pullRequestID+"/merge:"+helper.checkoutParam)
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

// SubmoduleUpdate ...
func (helper Helper) SubmoduleUpdate() error {
	cmdSlice, envs := createGitCmdSliceWithGitDontAskpass("submodule", "update", "--init", "--recursive")

	return runCommandInDirWithEnvs(cmdSlice, helper.destinationDir, append(os.Environ(), envs...))
}

func runLogCommand(cmdSlice []string, dir string) (string, error) {
	out, err := cmdex.NewCommand(cmdSlice[0], cmdSlice[1:]...).SetDir(dir).RunAndReturnTrimmedCombinedOutput()
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
