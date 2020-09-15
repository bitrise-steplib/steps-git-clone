package analytics

import (
	"os"

	"github.com/bitrise-io/go-utils/log"
)

type Data map[string]interface{}

// LogError sends analytics log using log.RErrorf by setting the stepID and data/build_slug.
func LogError(tag string, data Data, format string, v ...interface{}) {
	log.RErrorf("steps-git-clone", tag, data.appendSlug(), format, v...)
}

// LogWarn sends analytics log using log.RWarnf by setting the stepID and data/build_slug.
func LogWarn(tag string, data Data, format string, v ...interface{}) {
	log.RWarnf("steps-git-clone", tag, data.appendSlug(), format, v...)
}

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

func (a Data) AppendError(err error) Data {
	result := a
	if result == nil {
		result = CreateEmptyData()
	}
	result["error"] = err
	return result
}

func CreateEmptyData() Data {
	return map[string]interface{}{}
}
