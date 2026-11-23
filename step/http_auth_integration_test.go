package step

import (
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/stretchr/testify/require"
)

const testRepoPassword = "test-token"

func TestHTTPAuth_Integration(t *testing.T) {
	httpBackend := gitHTTPBackendPath(t)

	tests := []struct {
		name         string
		username     string
		password     string
		wantUsername string
		wantErr      bool
	}{
		{
			name:         "username and password",
			username:     "custom-user",
			password:     testRepoPassword,
			wantUsername: "custom-user",
		},
		{
			name:         "password without username",
			password:     testRepoPassword,
			wantUsername: "bitrise-git-clone-step",
		},
		{
			name:     "wrong password",
			username: "custom-user",
			password: "wrong-token",
			wantErr:  true,
		},
		{
			name:    "no credentials",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGitEnv(t)
			repoRoot, commit := createBareRepo(t)
			server := newAuthGitServer(t, httpBackend, repoRoot)

			cloneDir := t.TempDir()
			step := newIntegrationStep()
			_, err := step.Run(Config{Input: Input{
				RepositoryURL:   server.URL + "/repo.git",
				Branch:          "main",
				CloneIntoDir:    cloneDir,
				GitHTTPUsername: tt.username,
				GitHTTPPassword: tt.password,
			}})

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, commit, gitOutput(t, cloneDir, "rev-parse", "HEAD"))
			require.Equal(t, tt.wantUsername, server.lastUsername())
		})
	}
}

func newIntegrationStep() GitCloneStep {
	logger := log.NewLogger()
	envRepo := env.NewRepository()
	return NewGitCloneStep(GitCloneStepParams{
		Logger:       logger,
		Tracker:      tracker.NewStepTracker(envRepo, logger),
		InputParser:  stepconf.NewInputParser(envRepo),
		EnvRepo:      envRepo,
		CmdFactory:   command.NewFactory(envRepo),
		PathChecker:  pathutil.NewPathChecker(),
		PathModifier: pathutil.NewPathModifier(),
		Runner:       gitclone.NewDefaultRunner(logger),
	})
}

// isolateGitEnv keeps the developer's .netrc, git config and credential helpers out of the test,
// and keeps the .netrc written by the step out of the developer's home.
func isolateGitEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	t.Setenv("GIT_ASKPASS", "")
	t.Setenv("SSH_ASKPASS", "")
	t.Setenv("ANALYTICS_DISABLED", "true")
}

type authGitServer struct {
	*httptest.Server
	mu       sync.Mutex
	username string
}

func (s *authGitServer) lastUsername() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.username
}

// newAuthGitServer serves the repos under root with git's smart HTTP protocol, behind basic auth.
func newAuthGitServer(t *testing.T, httpBackend, root string) *authGitServer {
	backend := &cgi.Handler{
		Path: httpBackend,
		Env:  []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1"},
	}
	s := &authGitServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || password != testRepoPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		s.mu.Lock()
		s.username = username
		s.mu.Unlock()
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func createBareRepo(t *testing.T) (root, commit string) {
	root = t.TempDir()
	work := t.TempDir()
	gitOutput(t, work, "init", "-q", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(work, "README.md"), []byte("fixture\n"), 0644))
	gitOutput(t, work, "add", ".")
	gitOutput(t, work, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "Initial commit")
	gitOutput(t, root, "clone", "-q", "--bare", work, "repo.git")
	return root, gitOutput(t, work, "rev-parse", "HEAD")
}

func gitHTTPBackendPath(t *testing.T) string {
	execPath := gitOutput(t, "", "--exec-path")
	path := filepath.Join(execPath, "git-http-backend")
	if _, err := os.Stat(path); err != nil {
		// A skip would look like a pass in CI and hide that this test never ran.
		if os.Getenv("CI") != "" {
			t.Fatalf("git-http-backend not found: %v", err)
		}
		t.Skipf("git-http-backend not found: %v", err)
	}
	return path
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}
