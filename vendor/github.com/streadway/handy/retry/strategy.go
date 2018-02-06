package retry

import (
	"fmt"
	"io"
	"net"
	"time"
)

var DefaultRetryer = All(Max(10), Timeout(30*time.Second), EOF(), Over(300))

// All aggregates decisions from Retryers for an attempt.  All returns Abort
// and the error on the first Abort.  If at least one returns Retry All returns
// Retry with nil error.  Otherwise All returns Ignore with nil error.
func All(conditions ...Retryer) Retryer {
	return func(a Attempt) (Decision, error) {
		final := Ignore
		for _, eval := range conditions {
			decision, err := eval(a)

			switch decision {
			case Retry:
				final = Retry
			case Abort:
				return Abort, err
			}
		}
		return final, nil
	}
}

// "Forbidders" (return Abort or Ignore)

// TimeoutError is returned from RoundTrip when the time limit has been reached.
type TimeoutError struct {
	limit time.Duration
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("retry timed out after %s", e.limit)
}

// Timeout errors after a duration of time passes since the first attempt.
func Timeout(limit time.Duration) Retryer {
	return func(a Attempt) (Decision, error) {
		if time.Since(a.Start) >= limit {
			return Abort, TimeoutError{limit}
		}
		return Ignore, nil
	}
}

// MaxError is returned from RoundTrip when the maximum attempts has been reached.
type MaxError struct {
	limit uint
}

func (e MaxError) Error() string {
	return fmt.Sprintf("retry limit exceeded after %d attempts", e.limit)
}

// Max errors after a limited number of attempts
func Max(limit uint) Retryer {
	return func(a Attempt) (Decision, error) {
		if a.Count >= limit {
			return Abort, MaxError{limit}
		}
		return Ignore, nil
	}
}

// "Validators" (return Retry or Ignore)

// Errors returns Retry when the attempt produced an error.
func Errors() Retryer {
	return func(a Attempt) (Decision, error) {
		if a.Err != nil {
			return Retry, nil
		}
		return Ignore, nil
	}
}

// Net retries errors returned by the 'net' package.
func Net() Retryer {
	return func(a Attempt) (Decision, error) {
		if _, isNetError := a.Err.(*net.OpError); isNetError {
			return Retry, nil
		}
		return Ignore, nil
	}
}

// Temporary retries if the error implements Temporary() bool and returns true or aborts if returning false.
func Temporary() Retryer {
	type temper interface {
		Temporary() bool
	}
	return func(a Attempt) (Decision, error) {
		if t, ok := a.Err.(temper); ok {
			if t.Temporary() {
				return Retry, nil
			} else {
				return Abort, nil
			}
		}
		return Ignore, nil
	}
}

// EOF retries only when the error is EOF
func EOF() Retryer {
	return func(a Attempt) (Decision, error) {
		if a.Err == io.EOF {
			return Retry, nil
		}
		return Ignore, nil
	}
}

// Over retries when a response is missing or the status code is over a value like 300
func Over(statusCode int) Retryer {
	return func(a Attempt) (Decision, error) {
		if a.Response == nil {
			return Ignore, nil
		}
		if a.Response.StatusCode >= statusCode {
			return Retry, nil
		}
		return Ignore, nil
	}
}
