package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	t.Log("it doesn not retryies - if no error")
	{
		retryCnt := 0

		err := Times(2).Retry(func(attempt uint) error {
			retryCnt++
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 1, retryCnt)
	}

	t.Log("it retryies - if error")
	{
		retryCnt := 0
		actionErr := errors.New("error")

		err := Times(2).Retry(func(attempt uint) error {
			retryCnt++
			return actionErr
		})

		require.Error(t, err)
		require.Equal(t, "error", err.Error())
		require.Equal(t, 2, retryCnt)
	}

	t.Log("it does not wait before first execution")
	{
		retryCnt := 0
		actionErr := errors.New("error")
		startTime := time.Now()

		err := Times(1).Wait(10).Retry(func(attempt uint) error {
			retryCnt++
			return actionErr
		})

		duration := time.Now().Sub(startTime)

		require.Error(t, err)
		require.Equal(t, "error", err.Error())
		require.Equal(t, 1, retryCnt)
		if duration >= time.Duration(10)*time.Second {
			t.Fatalf("Should take no more than 10 sec, but got: %s", duration)
		}
	}
	t.Log("it waits before second execution")
	{
		retryCnt := 0
		actionErr := errors.New("error")
		startTime := time.Now()

		err := Times(2).Wait(10).Retry(func(attempt uint) error {
			retryCnt++
			return actionErr
		})

		duration := time.Now().Sub(startTime)

		require.Error(t, err)
		require.Equal(t, "error", err.Error())
		require.Equal(t, 2, retryCnt)
		if duration < time.Duration(10)*time.Second {
			t.Fatalf("Should take at least 10 sec, but got: %s", duration)
		}
	}
}

func TestWait(t *testing.T) {
	t.Log("it creates retry model with wait time")
	{
		helper := Wait(3)
		require.Equal(t, uint(3), helper.waitSec)
	}

	t.Log("it creates retry model with wait time")
	{
		helper := Wait(3)
		helper.Wait(5)
		require.Equal(t, uint(5), helper.waitSec)
	}
}

func TestTimes(t *testing.T) {
	t.Log("it creates retry model with retry times")
	{
		helper := Times(3)
		require.Equal(t, uint(3), helper.retry)
	}

	t.Log("it sets retry times")
	{
		helper := Times(3)
		helper.Times(5)
		require.Equal(t, uint(5), helper.retry)
	}
}
