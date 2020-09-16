package analytics

import (
	"github.com/bitrise-io/go-utils/log"
)

// LogError sends analytics log using log.RErrorf by setting the stepID and data/build_slug.
func LogError(tag string, err error, format string, v ...interface{}) {
	log.RErrorf("git-clone", tag, buildData(err), format, v...)
}

func buildData(err error) map[string]interface{} {
	data := map[string]interface{}{}
	data["source"] = "scanner"
	if err != nil {
		data["error"] = err
	}
	return data
}
