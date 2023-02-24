package bitriseapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/hashicorp/go-retryablehttp"
)

var ErrServiceAccountAuth = errors.New(`authentication error: Bitrise can't connect to the git server to check the freshness of the merge ref.
Check the Service Credential User in App Settings > Integrations`)

type MergeRefChecker interface {
	// IsMergeRefUpToDate returns true if the ref is safe to use in a checkout, and it reflects the latest state of the PR
	IsMergeRefUpToDate(ref string) (bool, error)
}

func NewMergeRefChecker(buildURL string, apiToken string, client *retryablehttp.Client, logger log.Logger, tracker tracker.StepTracker) MergeRefChecker {
	return apiMergeRefChecker{
		buildURL: buildURL,
		apiToken: apiToken,
		client:   client,
		logger:   logger,
		tracker:  tracker,
	}
}

type apiMergeRefChecker struct {
	buildURL string
	apiToken string
	client   *retryablehttp.Client
	logger   log.Logger
	tracker  tracker.StepTracker
}

type mergeRefResponse struct {
	Status string `json:"status"`
}

type mergeRefFetcher func(attempt uint) (mergeRefResponse, error)

func (c apiMergeRefChecker) IsMergeRefUpToDate(ref string) (bool, error) {
	if c.buildURL == "" {
		return false, fmt.Errorf("Bitrise build URL is not defined")
	}
	if c.apiToken == "" {
		return false, fmt.Errorf("Bitrise API token is not defined")
	}

	startTime := time.Now()
	isUpToDate, attempts, err := doPoll(c.fetchMergeRef, time.Second*2, c.logger)
	c.tracker.LogMergeRefCheck(time.Since(startTime), isUpToDate, attempts)
	return isUpToDate, err
}

func doPoll(fetcher mergeRefFetcher, retryWaitTime time.Duration, logger log.Logger) (bool, uint, error) {
	isUpToDate := false
	attempts := uint(0)
	pollAction := func(attempt uint) (err error, shouldAbort bool) {
		resp, err := fetcher(attempt)
		attempts = attempt + 1 // attempt is 0-indexed
		if err != nil {
			logger.Warnf("Error while checking merge ref: %s", err)
			logger.Warnf("Retrying request...")
			return err, false
		}
		switch resp.Status {
		case "up-to-date":
			isUpToDate = true
			logger.Donef("Attempt %d: merge ref is up-to-date", attempts)
			return nil, true
		case "auth_error": // TODO
			return ErrServiceAccountAuth, true
		case "pending":
			logger.Warnf("Attempt %d: not up-to-date yet", attempts)
			return fmt.Errorf("pending"), false
		default:
			logger.Warnf("Attempt %d: unknown status: %s", attempts, resp.Status)
			return fmt.Errorf("unknown status: %s", resp.Status), false
		}
	}

	err := retry.Times(5).Wait(retryWaitTime).TryWithAbort(pollAction)
	if err != nil {
		return false, attempts, err
	}

	return isUpToDate, attempts, nil
}

func (c apiMergeRefChecker) fetchMergeRef(attempt uint) (mergeRefResponse, error) {
	url := fmt.Sprintf("%s/pull_request_merge_ref_status", c.buildURL)
	req, err := retryablehttp.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return mergeRefResponse{}, err
	}
	req.Header.Set("HTTP_BUILD_API_TOKEN", c.apiToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return mergeRefResponse{}, err
	}
	defer resp.Body.Close()

	var response mergeRefResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return mergeRefResponse{}, err
	}

	return mergeRefResponse{}, nil
}
