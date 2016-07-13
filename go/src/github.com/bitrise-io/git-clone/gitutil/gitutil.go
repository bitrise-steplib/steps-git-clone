package gitutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	} else if branch != "" {
		helper.checkoutParam = branch
	}

	helper.cloneDepth = cloneDepth
}

// Init ...
func (helper Helper) Init() error {
	cmdSlice := createGitCmdSlice("init")
	return execute(helper.destinationDir, cmdSlice)
}

// RemoteAdd ...
func (helper Helper) RemoteAdd() error {
	cmdSlice, envs := createGitCmdSliceWithoutGitAskpass("remote", "add", "origin", helper.remoteURI)
	return executeWithEnvs(helper.destinationDir, envs, cmdSlice)
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

	cmdSlice, envs := createGitCmdSliceWithoutGitAskpass(params...)
	return executeWithEnvs(helper.destinationDir, envs, cmdSlice)
}

// ShouldCheckout ...
func (helper Helper) ShouldCheckout() bool {
	return (helper.checkoutParam != "")
}

// Checkout ...
func (helper Helper) Checkout() error {
	cmdSlice := createGitCmdSlice("checkout", helper.checkoutParam)
	return execute(helper.destinationDir, cmdSlice)
}

// ShouldTryFetchUnshallow ...
func (helper Helper) ShouldTryFetchUnshallow() bool {
	return (helper.cloneDepth != "")
}

// FetchUnshallow ...
func (helper Helper) FetchUnshallow() error {
	params := []string{"fetch", "--unshallow"}
	cmdSlice, envs := createGitCmdSliceWithoutGitAskpass(params...)
	return executeWithEnvs(helper.destinationDir, envs, cmdSlice)
}

// SubmoduleUpdate ...
func (helper Helper) SubmoduleUpdate() error {
	cmdSlice, envs := createGitCmdSliceWithoutGitAskpass("submodule", "update", "--init", "--recursive")
	return executeWithEnvs(helper.destinationDir, envs, cmdSlice)
}

// LogCommitHash ...
func (helper Helper) LogCommitHash() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%H"`)
	return executeForOutput(helper.destinationDir, cmdSlice)
}

// LogCommitMessageSubject ...
func (helper Helper) LogCommitMessageSubject() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%s"`)
	return executeForOutput(helper.destinationDir, cmdSlice)
}

// LogCommitMessageBody ...
func (helper Helper) LogCommitMessageBody() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%b"`)
	return executeForOutput(helper.destinationDir, cmdSlice)
}

// LogAuthorName ...
func (helper Helper) LogAuthorName() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%an"`)
	return executeForOutput(helper.destinationDir, cmdSlice)
}

// LogAuthorEmail ...
func (helper Helper) LogAuthorEmail() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%ae"`)
	return executeForOutput(helper.destinationDir, cmdSlice)
}

// LogCommiterName ...
func (helper Helper) LogCommiterName() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%cn"`)
	return executeForOutput(helper.destinationDir, cmdSlice)
}

// LogCommiterEmail ...
func (helper Helper) LogCommiterEmail() (string, error) {
	cmdSlice := createGitLogCmdSlice(`--format="%ce"`)
	return executeForOutput(helper.destinationDir, cmdSlice)
}

// ---------------------
//	Methods
// ---------------------

func createGitCmdSlice(params ...string) []string {
	return append([]string{"git"}, params...)
}

func createGitCmdSliceWithoutGitAskpass(params ...string) ([]string, []string) {
	return createGitCmdSlice(params...), []string{"GIT_ASKPASS=echo"}
}

func createGitLogCmdSlice(params ...string) []string {
	return append([]string{"git", "log", "-1"}, params...)
}

func properReturn(err error, out string) error {
	if err == nil {
		return nil
	}

	if errorutil.IsExitStatusError(err) {
		return errors.New(out)
	}
	return err
}

func execute(dir string, cmdSlice []string) error {
	return executeWithEnvs(dir, []string{}, cmdSlice)
}

func executeWithEnvs(dir string, envs []string, cmdSlice []string) error {
	if len(cmdSlice) == 0 {
		return errors.New("no command specified")
	}

	prinatableCmd := cmdex.PrintableCommandArgs(false, cmdSlice)
	log.Details("=> %s", prinatableCmd)

	out := ""
	var err error
	if len(cmdSlice) == 1 {
		out, err = run(dir, envs, cmdSlice[0])
	} else {
		out, err = run(dir, envs, cmdSlice[0], cmdSlice[1:len(cmdSlice)]...)
	}

	return properReturn(err, out)
}

func executeForOutput(dir string, cmdSlice []string) (string, error) {
	if len(cmdSlice) == 0 {
		return "", errors.New("no command specified")
	}

	prinatableCmd := cmdex.PrintableCommandArgs(false, cmdSlice)
	log.Details("=> %s", prinatableCmd)

	out := ""
	var err error
	if len(cmdSlice) == 1 {
		out, err = run(dir, []string{}, cmdSlice[0])
	} else {
		out, err = run(dir, []string{}, cmdSlice[0], cmdSlice[1:len(cmdSlice)]...)
	}

	if err != nil {
		return "", properReturn(err, out)
	}

	return out, nil
}

func run(dir string, envs []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(envs) > 0 {
		cmd.Env = append(os.Environ(), envs...)
	}
	outBytes, err := cmd.CombinedOutput()
	outStr := string(outBytes)
	return strings.TrimSpace(outStr), err
}
