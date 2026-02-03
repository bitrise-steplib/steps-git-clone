package gitclone

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
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
	perfEnv := r.performanceMonitoringEnvVar()
	c := t.Create(nil, nil, []string{perfEnv})

	fmt.Println()
	log.Infof("$ %s &> out", c.PrintableCommandArgs())

	out, err := c.RunAndReturnTrimmedCombinedOutput()
	if err != nil && errorutil.IsExitStatusError(err) {
		return out, errors.New(out)
	}

	return out, err
}

// Run ...
func (r *DefaultRunner) Run(t git.Template) error {
	var buffer bytes.Buffer

	perfEnv := r.performanceMonitoringEnvVar()
	c := t.Create(os.Stdout, io.MultiWriter(os.Stderr, &buffer), []string{perfEnv})

	fmt.Println()
	log.Infof("$ %s", c.PrintableCommandArgs())

	err := c.Run()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			errorStr := buffer.String()
			if errorStr == "" {
				errorStr = "please check the command output for errors"
			}
			return errors.New(strings.TrimSpace(errorStr))
		}
		return err
	}

	return nil
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

func (r *DefaultRunner) performanceMonitoringEnvVar() string {
	if r.performanceMonitoringTemporarilyDisabled {
		return "GIT_TRACE2_PERF=0"
	}

	if r.performanceMonitoringEnabled {
		return "GIT_TRACE2_PERF=1"
	}

	return ""
}
