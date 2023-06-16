package transport

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

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

	// Setup is a no-op if password is provided
	if cfg.HTTPPassword == "" {
		return nil
	}

	url, err := url.Parse(cfg.URL)
	if err != nil {
		return errors.Wrap(err, "failed to parse URL")
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

	// copied as a PoC from
	// https://github.com/bitrise-steplib/steps-authenticate-host-with-netrc/blob/master/main.go
	netRC.AddItemModel(netrcutil.NetRCItemModel{Machine: host, Login: username, Password: password})

	isExists, err := pathutil.IsPathExists(netRC.OutputPth)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to check path (%s)", netRC.OutputPth))
	}

	if !isExists {
		log.Printf("No .netrc file found at (%s), creating new...", netRC.OutputPth)

		if err := netRC.CreateFile(); err != nil {
			return errors.Wrap(err, "Failed to write .netrc file")
		}
	} else {
		log.Warnf(".netrc file already exists at (%s)", netRC.OutputPth)

		backupPth := fmt.Sprintf("%s%s", strings.Replace(netRC.OutputPth, ".netrc", ".bk.netrc", -1), time.Now().Format("2006_01_02_15_04_05"))

		if originalContent, err := fileutil.ReadBytesFromFile(netRC.OutputPth); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to read file (%s)", netRC.OutputPth))
		} else if err := fileutil.WriteBytesToFile(backupPth, originalContent); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to write file (%s)", backupPth))
		} else {
			log.Printf("Backup created at: %s", backupPth)
		}

		log.Printf("Appending config to the existing .netrc file...")

		if err := netRC.Append(); err != nil {
			return errors.Wrap(err, "Failed to write .netrc file")
		}
	}
	return nil
}
