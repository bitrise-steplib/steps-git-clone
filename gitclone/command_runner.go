package gitclone

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/git"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retry"
)

// CommandRunner ...
type CommandRunner interface {
	RunForOutput(t git.Template) (string, error)
	Run(t git.Template) error
	RunWithRetry(get func() git.Template) error
	SetPerformanceMonitoring(enable bool)
	PausePerformanceMonitoring()
	ResumePerformanceMonitoring()
}

// DefaultRunner ...
type DefaultRunner struct {
	logger                                   log.Logger
	performanceMonitoringEnabled             bool
	performanceMonitoringTemporarilyDisabled bool
}

func NewDefaultRunner(logger log.Logger) *DefaultRunner {
	return &DefaultRunner{
		logger: logger,
	}
}

// RunForOutput ...
func (r *DefaultRunner) RunForOutput(t git.Template) (string, error) {
	c := t.Create(nil, nil, r.performanceMonitoringEnvs())

	fmt.Println()
	r.logger.Infof("$ %s &> out", c.PrintableCommandArgs())

	out, err := c.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		var exitErr *command.ExitStatusError
		if errors.As(err, &exitErr) {
			return out, errors.New(out)
		}
	}

	return out, err
}

// Run ...
func (r *DefaultRunner) Run(t git.Template) error {
	var buffer bytes.Buffer

	c := t.Create(os.Stdout, io.MultiWriter(os.Stderr, &buffer), r.performanceMonitoringEnvs())

	fmt.Println()
	r.logger.Infof("$ %s", c.PrintableCommandArgs())

	err := c.Run()
	if err == nil {
		return nil
	}

	var exitErr *command.ExitStatusError
	if errors.As(err, &exitErr) {
		errorStr := strings.TrimSpace(buffer.String())
		if errorStr == "" {
			errorStr = "please check the command output for errors"
		}
		return errors.New(errorStr)
	}

	return err
}

// RunWithRetry ...
func (r *DefaultRunner) RunWithRetry(get func() git.Template) error {
	return retry.Times(2).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			r.logger.Warnf("Retrying...")
		}

		err := r.Run(get())
		if err != nil {
			r.logger.Warnf("Attempt %d failed:", attempt+1)
			fmt.Println(err.Error())
		}

		return err
	})
}

func (r *DefaultRunner) SetPerformanceMonitoring(enable bool) {
	r.performanceMonitoringEnabled = enable
}

func (r *DefaultRunner) PausePerformanceMonitoring() {
	r.performanceMonitoringTemporarilyDisabled = true
}

func (r *DefaultRunner) ResumePerformanceMonitoring() {
	r.performanceMonitoringTemporarilyDisabled = false
}

func (r *DefaultRunner) performanceMonitoringEnvs() []string {
	if r.performanceMonitoringTemporarilyDisabled {
		return []string{"GIT_TRACE2_PERF=0"}
	}

	if r.performanceMonitoringEnabled {
		return []string{"GIT_TRACE2_PERF=1"}
	}

	return nil
}
