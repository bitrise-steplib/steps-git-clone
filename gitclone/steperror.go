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
	var matcher *errormapper.PatternErrorMatcher
	switch tag {
	case checkoutFailedTag:
		matcher = newCheckoutFailedPatternErrorMatcher()
	case updateSubmodelFailedTag:
		matcher = newUpdateSubmoduleFailedErrorMatcher()
	case fetchFailedTag:
		matcher = newFetchFailedPatternErrorMatcher()
	}
	if matcher != nil {
		return matcher.Run(errMsg)
	}
	return nil
}

func newStepError(tag string, err error, shortMsg string) error {
	recommendations := mapDetailedErrorRecommendation(tag, err.Error())
	if recommendations != nil {
		return step.NewErrorWithRecommendations("git-clone", tag, err, shortMsg, recommendations)
	}

	return step.NewError("git-clone", tag, err, shortMsg)
}

func newStepErrorWithBranchRecommendations(tag string, err error, shortMsg, currentBranch string, availableBranches []string) error {
	// First: Map the error messages
	newErr := newStepError(tag, err, shortMsg)

	if mappedError, ok := newErr.(*step.Error); ok {
		// Second: Extend recommendation with available branches, if has any
		if len(availableBranches) > 0 {
			rec := mappedError.Recommendations
			if rec == nil {
				rec = step.Recommendation{}
			}
			rec[branchRecKey] = availableBranches
		}
	}

	return newErr
}

func newUpdateSubmoduleFailedErrorMatcher() *errormapper.PatternErrorMatcher {
	return &errormapper.PatternErrorMatcher{
		DefaultBuilder: newUpdateSubmoduleFailedGenericDetailedError,
		PatternToBuilder: errormapper.PatternToDetailedErrorBuilder{
			`ERROR: Repository not found`:                         newUpdateSubmoduleFailedAuthenticationDetailedError,
			`Invalid username or password`:                        newUpdateSubmoduleFailedAuthenticationDetailedError,
			`Permission denied`:                                   newUpdateSubmoduleFailedAuthenticationDetailedError,
			`HTTP Basic: Access denied`:                           newUpdateSubmoduleFailedAuthenticationDetailedError,
			`Unauthorized`:                                        newUpdateSubmoduleFailedAuthenticationDetailedError,
			`The project you were looking for could not be found`: newUpdateSubmoduleFailedAuthenticationDetailedError,
			`Unauthorized LoginAndPassword\(.+\)`:                 newUpdateSubmoduleFailedAuthenticationDetailedError,
		},
	}
}

func newUpdateSubmoduleFailedGenericDetailedError(errorMsg string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title: "We couldn’t update your submodules.",
		Description: fmt.Sprintf(`You can continue adding your app, but your builds will fail unless you fix the issue later.
Our auto-configurator returned the following error:
%s`, errorMsg),
	}
}

func newUpdateSubmoduleFailedAuthenticationDetailedError(errorMsg string, params ...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title: "We couldn’t access one or more of your Git submodules.",
		Description: fmt.Sprintf(`You can try accessing your submodules <a target="_blank" href="https://devcenter.bitrise.io/faq/adding-projects-with-submodules/">using an SSH key</a>. You can continue adding your app, but your builds will fail unless you fix this issue later.
Our auto-configurator returned the following error:
%s`, errorMsg),
	}
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

func newCheckoutFailedGenericDetailedError(errorMsg string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t checkout your branch.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error:\n%s", errorMsg),
	}
}

func newFetchFailedGenericDetailedError(errorMsg string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t fetch your repository.",
		Description: fmt.Sprintf("Our auto-configurator returned the following error:\n%s", errorMsg),
	}
}

func newFetchFailedSSHAccessErrorDetailedError(errorMsg string, params ...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t access your repository.",
		Description: "Please abort the process, double-check your SSH key and try again.",
	}
}

func newFetchFailedCouldNotFindGitRepoDetailedError(errorMsg string, params ...string) errormapper.DetailedError {
	repoURL := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       fmt.Sprintf("We couldn’t find a git repository at '%s'.", repoURL),
		Description: "Please abort the process, double-check your repository URL and try again.",
	}
}

func newFetchFailedHTTPAccessErrorDetailedError(errorMsg string, params ...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t access your repository.",
		Description: "Please abort the process and try again, by providing the repository with SSH URL.",
	}
}

func newFetchFailedCouldConnectErrorDetailedError(errorMsg string, params ...string) errormapper.DetailedError {
	host := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       fmt.Sprintf("We couldn’t connect to '%s'.", host),
		Description: "Please abort the process, double-check your repository URL and try again.",
	}
}

func newFetchFailedSamlSSOEnforcedDetailedError(errorMsg string, params ...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "To access this repository, you need to use SAML SSO.",
		Description: `Please abort the process, update your SSH settings and try again. You can find out more about <a target="_blank" href="https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/authorizing-an-ssh-key-for-use-with-saml-single-sign-on">using SAML SSO in the Github docs</a>.`,
	}
}

func newInvalidBranchDetailedError(errorMsg string, params ...string) errormapper.DetailedError {
	branch := errormapper.GetParamAt(0, params)
	return errormapper.DetailedError{
		Title:       fmt.Sprintf("We couldn't find the branch '%s'.", branch),
		Description: "Please choose another branch and try again.",
	}
}
