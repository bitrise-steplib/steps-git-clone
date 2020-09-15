package analytics

import (
	"os"

	"github.com/bitrise-io/go-utils/log"
)

// Data represents a map of properties sent with the analytics events
type Data map[string]interface{}

// LogError sends analytics log using log.RErrorf by setting the stepID and data/build_slug.
func LogError(tag string, data Data, format string, v ...interface{}) {
	log.RErrorf("steps-git-clone", tag, data.appendSlug(), format, v...)
}

// LogWarn sends analytics log using log.RWarnf by setting the stepID and data/build_slug.
func LogWarn(tag string, data Data, format string, v ...interface{}) {
	log.RWarnf("steps-git-clone", tag, data.appendSlug(), format, v...)
}

// appendSlug tries to append build slug to the analytics data
func (a Data) appendSlug() Data {
	result := a
	slug := os.Getenv("BITRISE_BUILD_SLUG")
	if slug == "" {
		return result
	}
	if result == nil {
		result = CreateEmptyData()
	}
	result["build_slug"] = slug
	return result
}

// AppendError appends error object to the analytics data
func (a Data) AppendError(err error) Data {
	result := a
	if result == nil {
		result = CreateEmptyData()
	}
	result["error"] = err
	return result
}

// CreateEmptyData initializes an empty data map
func CreateEmptyData() Data {
	return map[string]interface{}{}
}
