package bench

import (
	"encoding/base64"
	"fmt"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

var bitriseYMLTemplate = `
format_version: "8"
default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git

workflows:
  test:
    steps:
      - path::%s:
          run_if: true
          inputs:
          - clone_into_dir: "%s"
          - repository_url: "%s"
          - commit: "%s"
          - tag: "%s"
          - branch: "%s"
`

func bitriseRun(localStepPath, repositoryURL, commit, tag, branch string) error {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("git-clone")
	if err != nil {
		return fmt.Errorf("failed to create tmp dir: %s", err)
	}

	cloneIntoDir := tmpDir

	bitriseYML := fmt.Sprintf(bitriseYMLTemplate,
		localStepPath,
		cloneIntoDir,
		repositoryURL,
		commit,
		tag,
		branch,
	)

	// fmt.Println(bitriseYML)

	config := base64.StdEncoding.EncodeToString([]byte(bitriseYML))

	cmd := command.New("bitrise", "run", "test", "--config-base64", config)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
	}
	return nil
}
