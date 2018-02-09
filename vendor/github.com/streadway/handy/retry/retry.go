// Package retry implements a retrying transport based on a combination of strategies.
package retry

import (
	"net/http"
	"time"
)

var now = time.Now

// Attempt counts the round trips issued, starting from 1.  Response is valid
// only when Err is nil.
type Attempt struct {
	Start time.Time
	Count uint
	Err   error
	*http.Request
	*http.Response
}

// Delayer sleeps or selects any amount of time for each attempt.
type Delayer func(Attempt)

// Decision signals the intent of a Retryer
type Decision int

const (
	Ignore Decision = iota
	Retry
	Abort
)

// Retryer chooses whether or not to retry this request.  The error is only
// valid when the Retyer returns Abort.
type Retryer func(Attempt) (Decision, error)

type Transport struct {
	// Delay is called for attempts that are retried.  If nil, no delay will be used.
	Delay Delayer

	// Retry is called for every attempt
	Retry Retryer

	// Next is called for every attempt
	Next http.RoundTripper
}

// RoundTrip delegates a RoundTrip, then determines via Retry whether to retry
// and Delay for the wait time between attempts.
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		retryer = t.Retry
		start   = now()
	)
	if retryer == nil {
		retryer = DefaultRetryer
	}

	for count := uint(1); ; count++ {
		// Perform request
		resp, err := t.Next.RoundTrip(req)

		// Collect result of attempt
		attempt := Attempt{
			Start:    start,
			Count:    count,
			Err:      err,
			Request:  req,
			Response: resp,
		}

		// Evaluate attempt
		retry, retryErr := retryer(attempt)

		// Returns either the valid response or an error coming from the underlying Transport
		if retry == Ignore {
			return resp, err
		}

		// Close the response body when we wont use it anymore (Retry or Abort)
		if resp != nil {
			resp.Body.Close()
		}

		// Return the error explaining why we aborted and nil as response
		if retry == Abort {
			return nil, retryErr
		}

		// ... Retries (stay the loop)

		// Delay next attempt
		if t.Delay != nil {
			t.Delay(attempt)
		}
	}
	panic("unreachable")
}
