package gitclone

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/git"
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
	performanceMonitoringEnabled             bool
	performanceMonitoringTemporarilyDisabled bool
}

// RunForOutput ...
func (r *DefaultRunner) RunForOutput(t git.Template) (string, error) {
	c := t.Create(nil, nil, r.performanceMonitoringEnvs())

	fmt.Println()
	log.Infof("$ %s &> out", c.PrintableCommandArgs())

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
	log.Infof("$ %s", c.PrintableCommandArgs())

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
	return retry.Times(2).Wait(5).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("Retrying...")
		}

		err := r.Run(get())
		if err != nil {
			log.Warnf("Attempt %d failed:", attempt+1)
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
