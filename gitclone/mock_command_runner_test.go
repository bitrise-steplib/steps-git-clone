package gitclone

import (
	"errors"

	"github.com/bitrise-io/go-utils/command"
	"github.com/stretchr/testify/mock"
)

type MockRunner struct {
	mock.Mock
	cmds []string
}

// Cmds...
func (m *MockRunner) Cmds() []string {
	return m.cmds
}

// RunForOutput ...
func (m *MockRunner) RunForOutput(c *command.Model) (string, error) {
	args := m.Called(c)
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
func (m *MockRunner) Run(c *command.Model) error {
	args := m.Called(c)
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
	m.On("Run", mock.MatchedBy(func(command *command.Model) bool {
		return m.isCommandMatching(command, cmdString)
	})).
		Run(m.rememberCommand).
		Times(times).
		Return(errors.New("dummy_cmd_error"))
	return m
}

// RunWithRetry ...
func (m *MockRunner) RunWithRetry(getCommand func() *command.Model) error {
	args := m.Called(getCommand)
	return args.Error(0)
}

// GivenRunWithRetrySucceedsAfter ...
func (m *MockRunner) GivenRunWithRetrySucceeds() *MockRunner {
	return m.GivenRunWithRetrySucceedsAfter(0)
}

// GivenRunWithRetrySucceedsAfter ...
func (m *MockRunner) GivenRunWithRetrySucceedsAfter(times int) *MockRunner {
	m.On("RunWithRetry", mock.Anything).
		Run(func(args mock.Arguments) {
			for i := 0; i < times+1; i++ {
				m.rememberCommand(args)
			}
		}).
		Return(nil)
	return m
}

// GivenRunWithRetryFails ...
func (m *MockRunner) GivenRunWithRetryFailsAfter(times int) *MockRunner {
	m.On("RunWithRetry", mock.Anything).
		Run(func(args mock.Arguments) {
			for i := 0; i < times+1; i++ {
				m.rememberCommand(args)
			}
		}).
		Return(errors.New("dummy_cmd_error"))
	return m
}

// GivenRunWithRetryFails ...
func (m *MockRunner) GivenRunWithRetryFailsForCommand(cmdString string, times int) *MockRunner {
	m.On("RunWithRetry", mock.MatchedBy(func(command *command.Model) bool {
		return m.isCommandMatching(command, cmdString)
	})).
		Run(func(args mock.Arguments) {
			for i := 0; i < times+1; i++ {
				m.rememberCommand(args)
			}
		}).
		Return(errors.New("dummy_cmd_error"))
	return m
}

func (m *MockRunner) rememberCommand(args mock.Arguments) {
	var cmdModel *command.Model
	switch res := args[0].(type) {
	case *command.Model:
		cmdModel = res
	case func() *command.Model:
		cmdModel = res()
	}

	m.cmds = append(m.cmds, cmdModel.PrintableCommandArgs())
}

func (m *MockRunner) isCommandMatching(command *command.Model, cmdString string) bool {
	return command.PrintableCommandArgs() == cmdString
}
