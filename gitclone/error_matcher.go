package gitclone

import (
	"regexp"

	"github.com/bitrise-io/bitrise-init/step"
)

const (
	unknownParam        = "::unknown::"
	detailedErrorRecKey = "DetailedError"
)

// DetailedError ...
type DetailedError struct {
	Title       string
	Description string
}

func newDetailedErrorRecommendation(detailedError DetailedError) step.Recommendation {
	return step.Recommendation{
		detailedErrorRecKey: detailedError,
	}
}

// DetailedErrorBuilder ...
type DetailedErrorBuilder = func(...string) DetailedError

func getParamAt(index int, params []string) string {
	res := unknownParam
	if index >= 0 && len(params) > index {
		res = params[index]
	}
	return res
}

// PatternToDetailedErrorBuilder ...
type PatternToDetailedErrorBuilder map[string]DetailedErrorBuilder

// PatternErrorMatcher ...
type PatternErrorMatcher struct {
	defaultBuilder   DetailedErrorBuilder
	patternToBuilder PatternToDetailedErrorBuilder
}

func newPatternErrorMatcher(defaultBuilder DetailedErrorBuilder, patternToBuilder PatternToDetailedErrorBuilder) *PatternErrorMatcher {
	m := PatternErrorMatcher{
		defaultBuilder:   defaultBuilder,
		patternToBuilder: patternToBuilder,
	}

	return &m
}

// Run ...
func (m *PatternErrorMatcher) Run(msg string) step.Recommendation {
	for pattern, builder := range m.patternToBuilder {
		re := regexp.MustCompile(pattern)
		if re.MatchString(msg) {
			// [search_string, match1, match2, ...]
			matches := re.FindStringSubmatch((msg))
			// Drop the first item, which is always the search_string itself
			// [search_string] -> []
			// [search_string, match1, ...] -> [match1, ...]
			params := matches[1:]
			detail := builder(params...)
			return newDetailedErrorRecommendation(detail)
		}
	}

	detail := m.defaultBuilder(msg)
	return newDetailedErrorRecommendation(detail)
}
