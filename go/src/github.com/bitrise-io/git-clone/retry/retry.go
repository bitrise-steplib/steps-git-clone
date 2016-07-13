package retry

import (
	"fmt"
	"time"
)

// Action ...
type Action func(attempt uint) error

// RetryModel ...
type RetryModel struct {
	retry   uint
	waitSec uint
}

// Times ...
func Times(retry uint) *RetryModel {
	RetryModel := RetryModel{}
	return RetryModel.Times(retry)
}

// Times ...
func (RetryModel *RetryModel) Times(retry uint) *RetryModel {
	RetryModel.retry = retry
	return RetryModel
}

// Wait ...
func Wait(wait uint) *RetryModel {
	RetryModel := RetryModel{}
	return RetryModel.Wait(wait)
}

// Wait ...
func (RetryModel *RetryModel) Wait(waitSec uint) *RetryModel {
	RetryModel.waitSec = waitSec
	return RetryModel
}

// Retry ...
func (RetryModel RetryModel) Retry(action Action) error {
	if action == nil {
		return fmt.Errorf("no action specified")
	}

	if RetryModel.retry == 0 {
		return nil
	}

	var err error

	for attempt := uint(0); (0 == attempt || nil != err) && attempt < RetryModel.retry; attempt++ {
		if attempt > 0 && RetryModel.waitSec > 0 {
			time.Sleep(time.Duration(RetryModel.waitSec) * time.Second)
		}

		err = action(attempt)
	}

	return err
}
