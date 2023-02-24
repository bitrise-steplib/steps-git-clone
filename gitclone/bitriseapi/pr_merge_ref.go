package bitriseapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/hashicorp/go-retryablehttp"
)

var logger = log.NewLogger()
var ErrServiceAccountAuth = errors.New(`authentication error: Bitrise can't connect to the git server to check the freshness of the merge ref.
Check the Service Credential User in App Settings > Integrations`)

type MergeRefChecker interface {
	IsMergeRefUpToDate(ref string) (bool, error)
}

func NewMergeRefChecker(buildURL string, apiToken string, client *retryablehttp.Client) MergeRefChecker {
	return apiMergeRefChecker{
		buildURL: buildURL,
		apiToken: apiToken,
		client:   client,
	}
}

type apiMergeRefChecker struct {
	buildURL string
	apiToken string
	client   *retryablehttp.Client
}

type mergeRefResponse struct {
	Status string `json:"status"`
}

type mergeRefFetcher func(attempt uint) (mergeRefResponse, error)

func (c apiMergeRefChecker) IsMergeRefUpToDate(ref string) (bool, error) {
	return doPoll(c.fetchMergeRef, time.Second*2)
}

func doPoll(fetcher mergeRefFetcher, retryWaitTime time.Duration) (bool, error) {
	isUpToDate := false
	pollAction := func(attempt uint) (err error, shouldAbort bool) {
		resp, err := fetcher(attempt)
		if err != nil {
			logger.Warnf("Error while checking merge ref: %s", err)
			logger.Warnf("Retrying request...")
			return err, false
		}
		switch resp.Status {
		case "up-to-date":
			isUpToDate = true
			return nil, true
		case "auth_error": // TODO
			return ErrServiceAccountAuth, true
		case "pending":
			logger.Warnf("Attempt %d: not up-to-date yet", attempt)
			return fmt.Errorf("pending"), false
		default:
			return fmt.Errorf("unknown status: %s", resp.Status), false
		}
	}

	err := retry.Times(5).Wait(retryWaitTime).TryWithAbort(pollAction)
	if err != nil {
		return false, err
	}
	return isUpToDate, nil
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
