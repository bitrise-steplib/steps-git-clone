package analytics

import (
	"errors"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestDataBuild(t *testing.T) {
	t.Log("it creates complex data")
	{
		key := "BITRISE_BUILD_SLUG"
		var value string
		if value = os.Getenv(key); value != "" {
			_ = os.Unsetenv(key)
			defer func() { _ = os.Setenv(key, value) }()
		} else {
			defer func() { _ = os.Unsetenv(key) }()
		}
		_ = os.Setenv(key, "testSlug")
		data := CreateEmptyData().AppendError(errors.New("testError")).appendSlug()
		require.Equal(t, "testError", data["error"].(error).Error())
		require.Equal(t, "testSlug", data["build_slug"])
	}
}
