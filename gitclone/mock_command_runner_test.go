package gitclone

import (
	"errors"
	"io"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/git"

	"github.com/stretchr/testify/mock"
)

type MockRunner struct {
	mock.Mock
	cmds []string
}

var errDummy = errors.New("dummy_cmd_error")

// Cmds...
func (m *MockRunner) Cmds() []string {
	return m.cmds
}

// RunForOutput ...
func (m *MockRunner) RunForOutput(t git.Template) (string, error) {
	args := m.Called(t)
	return args.String(0), args.Error(1)
}

// GivenRunForOutputSucceeds ...
func (m *MockRunner) GivenRunForOutputSucceeds() *MockRunner {
	m.On("RunForOutput", mock.Anything).
		Run(m.rememberCommand).
		Return("whatever", nil)
	return m
}

// Run ...
func (m *MockRunner) Run(t git.Template) error {
	args := m.Called(t)
	return args.Error(0)
}

// GivenRunSucceeds ...
func (m *MockRunner) GivenRunSucceeds() *MockRunner {
	m.On("Run", mock.Anything).
		Run(m.rememberCommand).
		Return(nil)
	return m
}

// GivenRunFailsForCommand ...
func (m *MockRunner) GivenRunFailsForCommand(cmdString string, times int) *MockRunner {
	m.On("Run", mock.MatchedBy(func(t git.Template) bool {
		return m.isCommandMatching(t, cmdString)
	})).
		Run(m.rememberCommand).
		Times(times).
		Return(errDummy)
	return m
}

// RunWithRetry ...
func (m *MockRunner) RunWithRetry(getCommand func() git.Template) error {
	args := m.Called(getCommand)
	return args.Error(0)
}

// GivenRunWithRetrySucceeds ...
func (m *MockRunner) GivenRunWithRetrySucceeds() *MockRunner {
	return m.GivenRunWithRetrySucceedsAfter(0)
}

// GivenRunWithRetrySucceedsAfter ...
func (m *MockRunner) GivenRunWithRetrySucceedsAfter(times int) *MockRunner {
	m.On("RunWithRetry", mock.Anything).
		Run(func(args mock.Arguments) {
			m.rememberCommands(args, times)
		}).
		Return(nil)
	return m
}

// GivenRunWithRetryFails ...
func (m *MockRunner) GivenRunWithRetryFailsAfter(times int) *MockRunner {
	m.On("RunWithRetry", mock.Anything).
		Run(func(args mock.Arguments) {
			m.rememberCommands(args, times)
		}).
		Return(errDummy)
	return m
}

func (m *MockRunner) SetPerformanceMonitoring(enable bool) {
	_ = enable
}

func (m *MockRunner) PausePerformanceMonitoring() {
}

func (m *MockRunner) ResumePerformanceMonitoring() {
}

func (m *MockRunner) rememberCommand(args mock.Arguments) {
	_, printable := templateToCommand(args[0])
	m.cmds = append(m.cmds, printable)
}

func (m *MockRunner) rememberCommands(args mock.Arguments, times int) {
	for i := 0; i < times+1; i++ {
		m.rememberCommand(args)
	}
}

func (m *MockRunner) isCommandMatching(t git.Template, cmdString string) bool {
	_, printable := templateToCommand(t)
	return printable == cmdString
}

func templateToCommand(v any) (command.Command, string) {
	var t git.Template
	switch res := v.(type) {
	case git.Template:
		t = res
	case func() git.Template:
		t = res()
	default:
		panic("Could not cast argument to git.Template")
	}

	c := t.Create(io.Discard, io.Discard, nil)
	return c, c.PrintableCommandArgs()
}
