// Package file provides a public API for storing and retrieving a single
// file in the Bitrise Build Cache by an arbitrary key. It exists so Go-based
// steps can reuse the save-file / restore-file functionality without shelling
// out to the bitrise-build-cache CLI.
package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/google/uuid"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/build_cache/kv"
	configcommon "github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// HelperParams configures the file save/restore Helper.
type HelperParams struct {
	// Envs is the set of environment variables used to read auth config and
	// endpoint overrides. If nil, the current process environment is read.
	Envs map[string]string

	// EndpointURL overrides the build cache endpoint URL.
	// If empty, the URL is selected from Envs (BITRISE_BUILD_CACHE_ENDPOINT) or
	// falls back to the configured default.
	EndpointURL string

	// DebugLogging enables verbose debug output on the default logger.
	// Ignored when Logger is set.
	DebugLogging bool

	// Logger overrides the default logger. If nil, a default logger is created.
	Logger log.Logger

	// CommandFunc is used to run external commands when collecting cache
	// metadata (git, hostname, etc.). If nil, exec.Command is used.
	CommandFunc configcommon.CommandFunc
}

// ErrCacheNotFound is returned (wrapped) by Restore when no entry exists for
// the given key.
var ErrCacheNotFound = kv.ErrCacheNotFound

// ClientName is the kv.Client name reported for save-file / restore-file
// operations. Exported so the cmd layer can use the same identifier.
const ClientName = "file"

// Helper saves and restores a single file in the Bitrise Build Cache.
//
// The zero value is not usable — construct via NewHelper.
type Helper struct {
	logger      log.Logger
	envs        map[string]string
	endpointURL string
	commandFunc configcommon.CommandFunc
}

// NewHelper returns a Helper configured from params, applying defaults for any
// nil fields. It does not perform any network or filesystem I/O — that happens
// in Save / Restore.
func NewHelper(params HelperParams) *Helper {
	envs := params.Envs
	if envs == nil {
		envs = utils.AllEnvs()
	}

	logger := params.Logger
	if logger == nil {
		logger = log.NewLogger(log.WithDebugLog(params.DebugLogging))
	}

	commandFunc := params.CommandFunc
	if commandFunc == nil {
		commandFunc = defaultCommandFunc
	}

	return &Helper{
		logger:      logger,
		envs:        envs,
		endpointURL: params.EndpointURL,
		commandFunc: commandFunc,
	}
}

// Save uploads the contents of filePath into the Bitrise Build Cache under the
// given key. Returns an error if key or filePath is empty, the file cannot be
// read, or the upload fails.
func (h *Helper) Save(ctx context.Context, key, filePath string) error {
	if err := validateArgs(key, filePath); err != nil {
		return err
	}

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("stat file %q: %w", filePath, err)
	}

	h.logger.Infof("(i) Cache key: %s", key)
	h.logger.Infof("(i) File: %s", filePath)

	kvClient, err := h.newKVClient(ctx)
	if err != nil {
		return err
	}

	h.logger.TInfof("Uploading %s for key %s", filePath, key)
	if err := kvClient.UploadFileToBuildCache(ctx, filePath, key); err != nil {
		return fmt.Errorf("upload file to build cache: %w", err)
	}

	return nil
}

// Restore downloads the cache entry stored under key and writes it to filePath,
// creating any missing parent directories. Returns an error wrapping
// ErrCacheNotFound if no entry exists for the given key.
func (h *Helper) Restore(ctx context.Context, key, filePath string) error {
	if err := validateArgs(key, filePath); err != nil {
		return err
	}

	h.logger.Infof("(i) Cache key: %s", key)
	h.logger.Infof("(i) File: %s", filePath)

	kvClient, err := h.newKVClient(ctx)
	if err != nil {
		return err
	}

	h.logger.TInfof("Downloading %s for key %s", filePath, key)
	if err := kvClient.DownloadFileFromBuildCache(ctx, filePath, key); err != nil {
		if errors.Is(err, kv.ErrCacheNotFound) {
			return fmt.Errorf("no cache item found for key %q: %w", key, err)
		}

		return fmt.Errorf("download file from build cache: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Private — Helper internals
// ---------------------------------------------------------------------------

func (h *Helper) newKVClient(ctx context.Context) (*kv.Client, error) {
	authConfig, err := configcommon.ReadAuthConfigFromEnvironments(h.envs)
	if err != nil {
		return nil, fmt.Errorf("read auth config from environments: %w", err)
	}

	endpointURL := configcommon.SelectCacheEndpointURL(h.endpointURL, h.envs)
	h.logger.Debugf("Build Cache Endpoint URL: %s", endpointURL)

	host, insecureGRPC, err := kv.ParseURLGRPC(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint URL %q: %w", endpointURL, err)
	}

	client, err := kv.NewClient(kv.NewClientParams{
		UseInsecure:         insecureGRPC,
		Host:                host,
		DialTimeout:         5 * time.Second,
		ClientName:          ClientName,
		AuthConfig:          authConfig,
		Logger:              h.logger,
		CacheConfigMetadata: configcommon.NewMetadata(h.envs, h.commandFunc, h.logger),
		CacheOperationID:    uuid.NewString(),
	})
	if err != nil {
		return nil, fmt.Errorf("new kv client: %w", err)
	}

	if err := client.GetCapabilitiesWithRetry(ctx); err != nil {
		return nil, fmt.Errorf("get capabilities: %w", err)
	}

	return client, nil
}

// ---------------------------------------------------------------------------
// Private — package-level helpers
// ---------------------------------------------------------------------------

func validateArgs(key, filePath string) error {
	if key == "" {
		return errors.New("key must not be empty")
	}

	if filePath == "" {
		return errors.New("file path must not be empty")
	}

	return nil
}

func defaultCommandFunc(name string, v ...string) (string, error) {
	//nolint:noctx // configcommon.CommandFunc has no context parameter; callers can inject their own.
	output, err := exec.Command(name, v...).Output()

	return string(output), err
}
