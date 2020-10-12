package step

// Error is an error occuring top level in a step
type Error struct {
	StepID, Tag, ShortMsg string
	Err                   error
}

// NewError constructs a step.Error
func NewError(stepID, tag string, err error, shortMsg string) *Error {
	return &Error{
		StepID:   stepID,
		Tag:      tag,
		Err:      err,
		ShortMsg: shortMsg,
	}
}

func (e *Error) Error() string {
	return e.Err.Error()
}
