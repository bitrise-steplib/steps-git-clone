package gitclone

import (
	"io"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
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

			gitTemplate := mockTemplate{
				mockCommand: command.NewFactory(env.NewRepository()).Create("echo", []string{"hello"}, nil),
			}
			err := r.Run(&gitTemplate)
			require.NoError(t, err)

			value, ok := getEnv(gitTemplate.receivedEnvs, "GIT_TRACE2_PERF")

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

func getEnv(envs []string, key string) (string, bool) {
	for _, e := range envs {
		if strings.HasPrefix(e, key) {
			return strings.TrimPrefix(e, key+"="), true
		}
	}
	return "", false
}

type mockTemplate struct {
	receivedEnvs []string
	mockCommand  command.Command
}

func (m *mockTemplate) Create(stdOut, stdErr io.Writer, envs []string) command.Command {
	m.receivedEnvs = envs
	return m.mockCommand
}
