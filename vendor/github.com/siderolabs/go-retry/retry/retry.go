// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package retry provides generic action retry.
package retry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// RetryableFunc represents a function that can be retried.
type RetryableFunc func() error

// RetryableFuncWithContext represents a function that can be retried.
type RetryableFuncWithContext func(context.Context) error

func removeContext(f RetryableFunc) RetryableFuncWithContext {
	return func(context.Context) error {
		return f()
	}
}

// Retryer defines the requirements for retrying a function.
type Retryer interface {
	Retry(RetryableFunc) error
	RetryWithContext(context.Context, RetryableFuncWithContext) error
}

// Ticker defines the requirements for providing a clock to the retry logic.
type Ticker interface {
	Tick() time.Duration
	StopChan() <-chan struct{}
	Stop()
}

// ErrorSet represents a set of unique errors.
type ErrorSet struct { //nolint:errname
	errs []error

	mu sync.Mutex
}

func (e *ErrorSet) Error() string {
	if len(e.errs) == 0 {
		return ""
	}

	errString := fmt.Sprintf("%d error(s) occurred:", len(e.errs))
	for _, err := range e.errs {
		errString = fmt.Sprintf("%s\n\t%s", errString, err)
	}

	return errString
}

// Append adds the error to the set if the error is not already present.
func (e *ErrorSet) Append(err error) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.errs == nil {
		e.errs = []error{}
	}

	ok := false

	for _, existingErr := range e.errs {
		if err.Error() == existingErr.Error() {
			ok = true

			break
		}
	}

	if !ok {
		e.errs = append(e.errs, err)
	}

	return ok
}

// Is implements errors.Is.
func (e *ErrorSet) Is(err error) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, ee := range e.errs {
		if errors.Is(ee, err) {
			return true
		}
	}

	return false
}

// TimeoutError represents a timeout error.
type TimeoutError struct{}

func (TimeoutError) Error() string {
	return "timeout"
}

// IsTimeout reutrns if the provided error is a timeout error.
func IsTimeout(err error) bool {
	return errors.Is(err, TimeoutError{})
}

type expectedError struct{ error }

func (e expectedError) Unwrap() error {
	return e.error
}

type unexpectedError struct{ error }

func (e unexpectedError) Unwrap() error {
	return e.error
}

type retryer struct {
	options  *Options
	duration time.Duration
}

type ticker struct {
	options *Options
	rand    *rand.Rand
	s       chan struct{}
}

func (t ticker) Jitter() time.Duration {
	if int(t.options.Jitter) == 0 {
		return time.Duration(0)
	}

	if t.rand == nil {
		t.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return time.Duration(t.rand.Int63n(int64(t.options.Jitter)))
}

func (t ticker) StopChan() <-chan struct{} {
	return t.s
}

func (t ticker) Stop() {
	close(t.s)
}

// ExpectedError error represents an error that is expected by the retrying
// function. This error is ignored.
func ExpectedError(err error) error {
	if err == nil {
		return nil
	}

	return expectedError{err}
}

// ExpectedErrorf makes an expected error from given format and arguments.
func ExpectedErrorf(format string, a ...interface{}) error {
	return ExpectedError(fmt.Errorf(format, a...))
}

// UnexpectedError error represents an error that is unexpected by the retrying
// function. This error is fatal.
//
// Deprecated: all errors are unexpected by default, just return them.
func UnexpectedError(err error) error {
	if err == nil {
		return nil
	}

	return unexpectedError{err}
}

func retry(ctx context.Context, f RetryableFuncWithContext, d time.Duration, t Ticker, o *Options) error {
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	errs := &ErrorSet{}

	var timer *time.Timer

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		err := func() error {
			var attemptCtxCancel context.CancelFunc

			attemptCtx := ctx

			if o.AttemptTimeout != 0 {
				attemptCtx, attemptCtxCancel = context.WithTimeout(attemptCtx, o.AttemptTimeout)
				defer attemptCtxCancel()
			}

			return f(attemptCtx)
		}()

		if err == nil {
			return nil
		}

		if errors.Is(err, context.DeadlineExceeded) {
			err = TimeoutError{}

			select {
			case <-ctx.Done():
			default:
				// main context not canceled, continue retrying
				err = ExpectedError(err)
			}
		}

		exists := errs.Append(err)

		var expError expectedError

		if errors.As(err, &expError) {
			// retry expected errors
			if !exists && o.LogErrors {
				log.Printf("retrying error: %s", err)
			}
		} else {
			return errs
		}

		timer = time.NewTimer(t.Tick())

		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.DeadlineExceeded) {
				err = TimeoutError{}
			}

			errs.Append(err)

			return errs
		case <-t.StopChan():
			return nil
		case <-timer.C:
		}
	}
}
