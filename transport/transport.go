package transport

import (
	"fmt"
	"net/url"
	"strings"

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

	if err := netRC.CreateOrUpdateFile(netrcutil.NetRCItemModel{Machine: host, Login: username, Password: password}); err != nil {
		return fmt.Errorf("failed to update .netrc file: %w", err)
	}

	return nil
}
