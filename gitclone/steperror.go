package gitclone

import (
	"fmt"
	"regexp"

	"github.com/bitrise-io/bitrise-init/step"
)

// DetailBuilder ...
type DetailBuilder = func(...string) Detail

// PatternErrorMatcher ...
type PatternErrorMatcher struct {
	defaultHandler DetailBuilder
	handlers       map[string]DetailBuilder
}

func newPatternErrorMatcher(defaultHandler DetailBuilder, handlers map[string]DetailBuilder) *PatternErrorMatcher {
	m := PatternErrorMatcher{
		handlers:       handlers,
		defaultHandler: defaultHandler,
	}

	return &m
}

// Run ...
func (m *PatternErrorMatcher) Run(msg string) step.Recommendation {
	for pattern, handler := range m.handlers {
		re := regexp.MustCompile(pattern)
		if re.MatchString(msg) {
			matches := re.FindStringSubmatch((msg))

			if len(matches) > 1 {
				matches = matches[1:]
			}

			if matches != nil {
				detail := handler(matches...)
				return newDetailedErrorRecommendation(detail)
			}
		}
	}

	detail := m.defaultHandler(msg)
	return newDetailedErrorRecommendation(detail)
}

func newCheckoutFailedPatternErrorMatcher() *PatternErrorMatcher {
	return newPatternErrorMatcher(
		checkoutFailedGenericDetail,
		map[string]DetailBuilder{},
	)
}

func newFetchFailedPatternErrorMatcher() *PatternErrorMatcher {
	return newPatternErrorMatcher(
		fetchFailedGenericDetail,
		map[string]DetailBuilder{
			`Permission denied (.+)\.`:                                                               fetchFailedSSHAccessError,
			`fatal: repository '(.+)' not found`:                                                     fetchFailedCouldNotFindGitRepository,
			`fatal: '(.+)' does not appear to be a git repository`:                                   fetchFailedCouldNotFindGitRepository,
			`fatal: (.+)/info/refs not valid: is this a git repository?`:                             fetchFailedCouldNotFindGitRepository,
			`remote: HTTP Basic: Access denied[\n]*fatal: Authentication failed for '(.+)'`:          fetchFailedHTTPAccessError,
			`remote: Invalid username or password\(\.\)[\n]*fatal: Authentication failed for '(.+)'`: fetchFailedHTTPAccessError,
			`Unauthorized`:                          fetchFailedHTTPAccessError,
			`Forbidden`:                             fetchFailedHTTPAccessError,
			`remote: Unauthorized LoginAndPassword`: fetchFailedHTTPAccessError,
			// `fatal: unable to access '(.+)': Failed to connect to .+ port \d+: Connection timed out
			// `fatal: unable to access '(.+)': The requested URL returned error: 400`
			// `fatal: unable to access '(.+)': The requested URL returned error: 403`
			`fatal: unable to access '(.+)': (Failed|The requested URL returned error: \d+)`: fetchFailedHTTPAccessError,
			// `ssh: connect to host (.+) port \d+: Connection timed out`
			// `ssh: connect to host (.+) port \d+: Connection refused`
			// `ssh: connect to host (.+) port \d+: Network is unreachable`
			`ssh: connect to host (.+) port \d+:`:                                fetchFailedCouldConnectError,
			`ssh: Could not resolve hostname (.+): Name or service not known`:    fetchFailedCouldConnectError,
			`fatal: unable to access '.+': Could not resolve host: (\S+)`:        fetchFailedCouldConnectError,
			`ERROR: The \x60(.+)' organization has enabled or enforced SAML SSO`: fetchFailedSamlSSOError,
		})
}

func mapRecommendation(tag string, err error) step.Recommendation {
	msg := err.Error()
	switch tag {
	case checkoutFailedTag:
		matcher := newCheckoutFailedPatternErrorMatcher()
		return matcher.Run(msg)
	case updateSubmodelFailedTag: // update_submodule_failed could have the same errors as fetch
		fallthrough
	case fetchFailedTag:
		fetchFailedMatcher := newFetchFailedPatternErrorMatcher()
		return fetchFailedMatcher.Run(msg)
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

func checkoutFailedGenericDetail(params ...string) Detail {
	err := params[0]
	return Detail{
		Title:       "We couldn’t checkout your branch.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error:\n%s", err),
	}
}

func fetchFailedGenericDetail(params ...string) Detail {
	err := params[0]
	return Detail{
		Title:       "We couldn’t fetch your repository.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error:\n%s", err),
	}
}

func fetchFailedSSHAccessError(params ...string) Detail {
	return Detail{
		Title:       "We couldn’t access your repository.",
		Description: "Please abort the process, double-check your SSH key and try again.",
	}
}

func fetchFailedCouldNotFindGitRepository(params ...string) Detail {
	repoURL := params[0]
	return Detail{
		Title:       fmt.Sprintf("We couldn’t find a git repository at '%s'.", repoURL),
		Description: "Please abort the process, double-check your repository URL and try again.",
	}
}

func fetchFailedHTTPAccessError(params ...string) Detail {
	return Detail{
		Title:       "We couldn’t access your repository.",
		Description: "Please abort the process and try again, by providing the repository with SSH URL.",
	}
}

func fetchFailedCouldConnectError(params ...string) Detail {
	host := params[0]
	return Detail{
		Title:       fmt.Sprintf("We couldn’t connect to '%s'.", host),
		Description: "Please abort the process, double-check your repository URL and try again.",
	}
}

func fetchFailedSamlSSOError(params ...string) Detail {
	return Detail{
		Title:       "To access this repository, you need to use SAML SSO.",
		Description: `Please abort the process, update your SSH settings and try again. You can find out more about <a target="_blank" href="https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/authorizing-an-ssh-key-for-use-with-saml-single-sign-on">using SAML SSO in the Github docs</a>.`,
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
