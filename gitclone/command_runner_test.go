package gitclone

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/command"
	"github.com/stretchr/testify/require"
)

func TestPerformanceMonitoring(t *testing.T) {
	tests := []struct {
		name          string
		initialState  *bool
		shouldDisable bool
		want          *string
	}{
		{
			name:         "Normal case",
			initialState: nil,
			want:         nil,
		},
		{
			name:         "Enable performance monitoring",
			initialState: pointer(true),
			want:         pointer("1"),
		},
		{
			name:         "Disable performance monitoring",
			initialState: pointer(false),
		},
		{
			name:          "Temporarily disable performance monitoring",
			initialState:  pointer(true),
			shouldDisable: true,
			want:          pointer("0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := DefaultRunner{}

			if tt.initialState != nil {
				r.SetPerformanceMonitoring(*tt.initialState)
			}

			if tt.shouldDisable {
				r.PausePerformanceMonitoring()
			}

			cmd := command.New("echo", "hello")
			err := r.Run(cmd)
			require.NoError(t, err)

			value, ok := getEnv(cmd.GetCmd(), "GIT_TRACE2_PERF")

			if tt.want == nil {
				require.Equal(t, "", value)
				require.False(t, ok)
				return
			}

			if tt.shouldDisable {
				require.Equal(t, "0", value)
				require.True(t, ok)
				return
			}

			require.Equal(t, *tt.want, value)
			require.True(t, ok)
		})
	}
}

func pointer[T any](d T) *T {
	return &d
}

func getEnv(cmd *exec.Cmd, key string) (string, bool) {
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, key) {
			return strings.TrimPrefix(env, key+"="), true
		}
	}
	return "", false
}
