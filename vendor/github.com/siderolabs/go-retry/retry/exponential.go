// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package retry

import (
	"context"
	"math"
	"time"
)

type exponentialRetryer struct {
	retryer
}

// ExponentialTicker represents a ticker with a truncated exponential algorithm.
// Please see https://en.wikipedia.org/wiki/Exponential_backoff for details on
// the algorithm.
type ExponentialTicker struct {
	ticker

	c float64
}

// Exponential initializes and returns a truncated exponential Retryer.
func Exponential(duration time.Duration, setters ...Option) Retryer {
	opts := NewDefaultOptions(setters...)

	return exponentialRetryer{
		retryer: retryer{
			duration: duration,
			options:  opts,
		},
	}
}

// NewExponentialTicker is a ticker that sends the time on a channel using a
// truncated exponential algorithm.
func NewExponentialTicker(opts *Options) *ExponentialTicker {
	e := &ExponentialTicker{
		ticker: ticker{
			options: opts,
			s:       make(chan struct{}),
		},
		c: 1.0,
	}

	return e
}

// Retry implements the Retryer interface.
func (e exponentialRetryer) Retry(f RetryableFunc) error {
	return e.RetryWithContext(context.Background(), removeContext(f))
}

// RetryWithContext implements the Retryer interface.
func (e exponentialRetryer) RetryWithContext(ctx context.Context, f RetryableFuncWithContext) error {
	tick := NewExponentialTicker(e.options)
	defer tick.Stop()

	return retry(ctx, f, e.duration, tick, e.options)
}

// Tick implements the Ticker interface.
func (e *ExponentialTicker) Tick() time.Duration {
	d := time.Duration((math.Pow(2, e.c)-1)/2)*e.options.Units + e.Jitter()
	e.c++

	return d
}
