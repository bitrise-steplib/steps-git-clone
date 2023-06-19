package transport

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-authenticate-host-with-netrc/netrcutil"
)

type Config struct {
	URL          string
	HTTPUsername string
	HTTPPassword string
}

func Setup(cfg Config) error {
	// We only deal with http URLs for now
	if !strings.HasPrefix(cfg.URL, "http") {
		return nil
	}

	// Setup is a no-op if no password is provided
	if cfg.HTTPPassword == "" {
		return nil
	}

	url, err := url.Parse(cfg.URL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	host := url.Host
	username := cfg.HTTPUsername
	// Some providers (e.g. GitHub) doesn't care about the username, so we don't ask for it from the user
	// But something still needs to be provided when making the network call
	if username == "" {
		username = "bitrise-git-clone-step"
	}
	password := cfg.HTTPPassword

	netRC := netrcutil.New()

	// based on https://github.com/bitrise-steplib/steps-authenticate-host-with-netrc/blob/master/main.go
	netRC.AddItemModel(netrcutil.NetRCItemModel{Machine: host, Login: username, Password: password})

	isExists, err := pathutil.IsPathExists(netRC.OutputPth)
	if err != nil {
		return fmt.Errorf("failed to check path (%s): %w", netRC.OutputPth, err)
	}

	if !isExists {
		log.Debugf("No .netrc file found at (%s), creating new...", netRC.OutputPth)

		if err := netRC.CreateFile(); err != nil {
			return fmt.Errorf("failed to create .netrc file: %w", err)
		}
	} else {
		log.Warnf(".netrc file already exists at (%s)", netRC.OutputPth)

		backupPth := fmt.Sprintf("%s%s", strings.Replace(netRC.OutputPth, ".netrc", ".bk.netrc", -1), time.Now().Format("2006_01_02_15_04_05"))

		if originalContent, err := fileutil.ReadBytesFromFile(netRC.OutputPth); err != nil {
			return fmt.Errorf("failed to read file (%s): %w", netRC.OutputPth, err)
		} else if err := fileutil.WriteBytesToFile(backupPth, originalContent); err != nil {
			return fmt.Errorf("failed to write file (%s): %w", backupPth, err)
		} else {
			log.Warnf("Backup created at: %s", backupPth)
		}

		log.Debugf("Appending config to the existing .netrc file...")

		if err := netRC.Append(); err != nil {
			return fmt.Errorf("failed to append to .netrc file: %w", err)
		}
	}

	return nil
}
