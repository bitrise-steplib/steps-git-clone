package gitclone

import (
	"fmt"
	"regexp"

	"github.com/bitrise-io/bitrise-init/step"
)

func mapRecommendation(tag string, err error) step.Recommendation {
	switch tag {
	case "checkout_failed":
		detail := checkotFailedGenericDetail(err)
		return newDetailedErrorRecommendation(detail)
	case "fetch_failed":
		switch {
		case regexp.MustCompile(`Permission denied (.+)\.`).MatchString(err.Error()):
			detail := fetchFailedPermissionDeniedDetail()
			return newDetailedErrorRecommendation(detail)
		case regexp.MustCompile(`fatal: repository '(.+)' not found`).MatchString(err.Error()):
			matches := regexp.MustCompile(`fatal: repository '(.+)' not found`).FindStringSubmatch(err.Error())
			if len(matches) < 2 {
				break
			}
			repoURL := matches[1]

			detail := fetchFailedNotGitRepository(repoURL)
			return newDetailedErrorRecommendation(detail)
		default:
			detail := fetchFailedGenericDetail(err)
			return newDetailedErrorRecommendation(detail)
		}
	}
	return nil
}

func newStepError(tag string, err error, shortMsg string) *step.Error {
	recommendation := mapRecommendation(tag, err)
	if recommendation != nil {
		return step.NewErrorWithRecommendations("git-clone", tag, err, shortMsg, recommendation)
	}

	return step.NewError("git-clone", tag, err, shortMsg)
}

func newDetailedErrorRecommendation(detail Detail) step.Recommendation {
	return step.Recommendation{
		"DetailedError": map[string]string{
			"Title":       detail.Title,
			"Description": detail.Description,
		},
	}
}

func checkotFailedGenericDetail(err error) Detail {
	return Detail{
		Title:       "We couldn’t checkout your branch.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error:\n%s", err),
	}
}

func fetchFailedGenericDetail(err error) Detail {
	return Detail{
		Title:       "We couldn’t fetch your repository.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error: %s", err),
	}
}

func fetchFailedPermissionDeniedDetail() Detail {
	return Detail{
		Title:       "We couldn’t access your repository.",
		Description: "Please abort the process, double-check your SSH key and try again.",
	}
}

func fetchFailedNotGitRepository(repoURL string) Detail {
	return Detail{
		Title:       fmt.Sprintf("We couldn’t find a git repository at %s", repoURL),
		Description: "Please abort the process, double-check your repository URL and try again.",
	}
}

func newStepErrorWithRecommendations(tag string, err error, shortMsg string, recommendations step.Recommendation) *step.Error {
	return step.NewErrorWithRecommendations("git-clone", tag, err, shortMsg, recommendations)
}

// Detail ...
type Detail struct {
	Title       string
	Description string
}
