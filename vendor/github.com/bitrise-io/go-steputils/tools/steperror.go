package tools

// StepError is an error occuring top level in a step
type StepError struct {
	StepID, Tag, ShortMsg string
	Err                   error
}

// NewStepError constructs a StepError
func NewStepError(stepID, tag string, err error, shortMsg string) *StepError {
	return &StepError{
		StepID:   stepID,
		Tag:      tag,
		Err:      err,
		ShortMsg: shortMsg,
	}
}

func (e *StepError) Error() string {
	return e.Err.Error()
}
