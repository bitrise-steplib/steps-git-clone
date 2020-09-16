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
		var err error
		if value = os.Getenv(key); value != "" {
			err = os.Unsetenv(key)
			defer func() { err = os.Setenv(key, value) }()
		} else {
			defer func() { err = os.Unsetenv(key) }()
		}
		err = os.Setenv(key, "testSlug")
		require.Equal(t, nil, err)
		data := buildData(errors.New("testError"))
		require.Equal(t, "testError", data["error"].(error).Error())
		require.Equal(t, "scanner", data["source"])
	}
}
