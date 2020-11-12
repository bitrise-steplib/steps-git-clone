package gitclone

import (
	"fmt"

	"github.com/bitrise-io/bitrise-init/errormapper"
	"github.com/bitrise-io/bitrise-init/step"
)

const (
	branchRecKey = "BranchRecommendation"
)

func mapDetailedErrorRecommendation(tag, errMsg string) step.Recommendation {
	switch tag {
	case checkoutFailedTag:
		matcher := newCheckoutFailedPatternErrorMatcher()
		return matcher.Run(errMsg)
	case updateSubmodelFailedTag: // update_submodule_failed could have the same errors as fetch
		fallthrough
	case fetchFailedTag:
		matcher := newFetchFailedPatternErrorMatcher()
		return matcher.Run(errMsg)
	}
	return nil
}

func newStepError(tag string, err error, shortMsg string) *step.Error {
	recommendations := mapDetailedErrorRecommendation(tag, err.Error())
	if recommendations != nil {
		return step.NewErrorWithRecommendations("git-clone", tag, err, shortMsg, recommendations)
	}

	return step.NewError("git-clone", tag, err, shortMsg)
}

func newStepErrorWithBranchRecommendations(tag string, err error, shortMsg, currentBranch string, availableBranches []string) *step.Error {
	// First: Map the error messages
	mappedError := newStepError(tag, err, shortMsg)

	// Second: Extend recommendation with available branches, if has any
	if len(availableBranches) > 0 {
		rec := mappedError.Recommendations
		if rec == nil {
			rec = step.Recommendation{}
		}
		rec[branchRecKey] = availableBranches
	}

	return mappedError
}

func newCheckoutFailedPatternErrorMatcher() *errormapper.PatternErrorMatcher {
	return &errormapper.PatternErrorMatcher{
		DefaultBuilder: newCheckoutFailedGenericDetailedError,
		PatternToBuilder: errormapper.PatternToDetailedErrorBuilder{
			`pathspec '(.+)' did not match any file\(s\) known to git`: newInvalidBranchDetailedError,
		},
	}
}

func newFetchFailedPatternErrorMatcher() *errormapper.PatternErrorMatcher {
	return &errormapper.PatternErrorMatcher{
		DefaultBuilder: newFetchFailedGenericDetailedError,
		PatternToBuilder: errormapper.PatternToDetailedErrorBuilder{
			`Permission denied \((.+)\)`:                                                             newFetchFailedSSHAccessErrorDetailedError,
			`fatal: repository '(.+)' not found`:                                                     newFetchFailedCouldNotFindGitRepoDetailedError,
			`fatal: '(.+)' does not appear to be a git repository`:                                   newFetchFailedCouldNotFindGitRepoDetailedError,
			`fatal: (.+)/info/refs not valid: is this a git repository?`:                             newFetchFailedCouldNotFindGitRepoDetailedError,
			`remote: HTTP Basic: Access denied[\n]*fatal: Authentication failed for '(.+)'`:          newFetchFailedHTTPAccessErrorDetailedError,
			`remote: Invalid username or password\(\.\)[\n]*fatal: Authentication failed for '(.+)'`: newFetchFailedHTTPAccessErrorDetailedError,
			`Unauthorized`:                          newFetchFailedHTTPAccessErrorDetailedError,
			`Forbidden`:                             newFetchFailedHTTPAccessErrorDetailedError,
			`remote: Unauthorized LoginAndPassword`: newFetchFailedHTTPAccessErrorDetailedError,
			// `fatal: unable to access '(.+)': Failed to connect to .+ port \d+: Connection timed out
			// `fatal: unable to access '(.+)': The requested URL returned error: 400`
			// `fatal: unable to access '(.+)': The requested URL returned error: 403`
			`fatal: unable to access '(.+)': (Failed|The requested URL returned error: \d+)`: newFetchFailedHTTPAccessErrorDetailedError,
			// `ssh: connect to host (.+) port \d+: Connection timed out`
			// `ssh: connect to host (.+) port \d+: Connection refused`
			// `ssh: connect to host (.+) port \d+: Network is unreachable`
			`ssh: connect to host (.+) port \d+:`:                                newFetchFailedCouldConnectErrorDetailedError,
			`ssh: Could not resolve hostname (.+): Name or service not known`:    newFetchFailedCouldConnectErrorDetailedError,
			`fatal: unable to access '.+': Could not resolve host: (\S+)`:        newFetchFailedCouldConnectErrorDetailedError,
			`ERROR: The \x60(.+)' organization has enabled or enforced SAML SSO`: newFetchFailedSamlSSOEnforcedDetailedError,
		},
	}
}

func newCheckoutFailedGenericDetailedError(params ...string) errormapper.DetailedError {
	err := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       "We couldn’t checkout your branch.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error:\n%s", err),
	}
}

func newFetchFailedGenericDetailedError(params ...string) errormapper.DetailedError {
	err := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       "We couldn’t fetch your repository.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error:\n%s", err),
	}
}

func newFetchFailedSSHAccessErrorDetailedError(...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t access your repository.",
		Description: "Please abort the process, double-check your SSH key and try again.",
	}
}

func newFetchFailedCouldNotFindGitRepoDetailedError(params ...string) errormapper.DetailedError {
	repoURL := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       fmt.Sprintf("We couldn’t find a git repository at '%s'.", repoURL),
		Description: "Please abort the process, double-check your repository URL and try again.",
	}
}

func newFetchFailedHTTPAccessErrorDetailedError(...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t access your repository.",
		Description: "Please abort the process and try again, by providing the repository with SSH URL.",
	}
}

func newFetchFailedCouldConnectErrorDetailedError(params ...string) errormapper.DetailedError {
	host := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       fmt.Sprintf("We couldn’t connect to '%s'.", host),
		Description: "Please abort the process, double-check your repository URL and try again.",
	}
}

func newFetchFailedSamlSSOEnforcedDetailedError(...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "To access this repository, you need to use SAML SSO.",
		Description: `Please abort the process, update your SSH settings and try again. You can find out more about <a target="_blank" href="https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/authorizing-an-ssh-key-for-use-with-saml-single-sign-on">using SAML SSO in the Github docs</a>.`,
	}
}

func newInvalidBranchDetailedError(params ...string) errormapper.DetailedError {
	branch := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       fmt.Sprintf("We couldn't find the branch '%s'.", branch),
		Description: "Please choose another branch and try again.",
	}
}
