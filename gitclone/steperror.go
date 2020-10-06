package gitclone

import "fmt"

// StepError is an error occuring top level in a step
type StepError struct {
	Tag, ShortMsg string
	Err           error
}

func (e *StepError) Error() string {
	return fmt.Sprintf("%s, %s", e.ShortMsg, e.Err.Error())
}
