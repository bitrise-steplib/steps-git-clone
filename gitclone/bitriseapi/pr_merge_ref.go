package bitriseapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/hashicorp/go-retryablehttp"
)

const maxAttemptCount = 5
const retryWaitTime = time.Second * 2

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
	Status      string `json:"status"`
	Error       string `json:"error_msg"`
	ShouldRetry bool   `json:"should_retry"`
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
	isUpToDate, attempts, err := doPoll(c.fetchMergeRef, maxAttemptCount, retryWaitTime, c.logger)
	c.tracker.LogMergeRefVerify(time.Since(startTime), isUpToDate, attempts)
	return isUpToDate, err
}

func doPoll(fetcher mergeRefFetcher, maxAttemptCount uint, retryWaitTime time.Duration, logger log.Logger) (bool, uint, error) {
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
		if resp.Error != "" {
			// Soft-error from API
			logger.Warnf(resp.Error)
			logger.Warnf("Check your connected account in App settings > Integrations > Service credential user")
			return fmt.Errorf("response: %s", resp.Error), !resp.ShouldRetry
		}

		switch resp.Status {
		case "up-to-date":
			isUpToDate = true
			logger.Donef("Attempt %d: merge ref is up-to-date", attempts)
			return nil, true
		case "pending":
			logger.Warnf("Attempt %d: not up-to-date yet", attempts)
			return fmt.Errorf("pending"), false
		case "not-mergeable":
			// A not-mergeable PR state doesn't trigger a build directly, but there is a time window between
			// triggering a build from a mergeable PR and running this code, where the PR can become unmergeable.
			// The API responds with `not-mergeable` in this case.
			//
			// [PR push]-----[trigger]-------------------------------------------------[step run]
			// |---------------[PR push: merge conflict]-----[trigger: skip build]--------------|
			return fmt.Errorf("PR is not in a mergeable state"), true
		default:
			logger.Warnf("Attempt %d: unknown status: %s", attempts, resp)
			return fmt.Errorf("unknown status: %s", resp.Status), false
		}
	}

	err := retry.Times(maxAttemptCount).Wait(retryWaitTime).TryWithAbort(pollAction)
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
	req.Header.Set("BUILD_API_TOKEN", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return mergeRefResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warnf("Response status: %s", resp.Status)
	}

	var response mergeRefResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return mergeRefResponse{}, err
	}

	return response, nil
}
