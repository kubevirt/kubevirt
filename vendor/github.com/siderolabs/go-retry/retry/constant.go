// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package retry

import (
	"context"
	"time"
)

type constantRetryer struct {
	retryer
}

// ConstantTicker represents a ticker with a constant algorithm.
type ConstantTicker struct {
	ticker
}

// Constant initializes and returns a constant Retryer.
func Constant(duration time.Duration, setters ...Option) Retryer {
	opts := NewDefaultOptions(setters...)

	return constantRetryer{
		retryer: retryer{
			duration: duration,
			options:  opts,
		},
	}
}

// NewConstantTicker is a ticker that sends the time on a channel using a
// constant algorithm.
func NewConstantTicker(opts *Options) *ConstantTicker {
	l := &ConstantTicker{
		ticker: ticker{
			options: opts,
			s:       make(chan struct{}),
		},
	}

	return l
}

// Retry implements the Retryer interface.
func (c constantRetryer) Retry(f RetryableFunc) error {
	return c.RetryWithContext(context.Background(), removeContext(f))
}

// RetryWithContext implements the Retryer interface.
func (c constantRetryer) RetryWithContext(ctx context.Context, f RetryableFuncWithContext) error {
	tick := NewConstantTicker(c.options)
	defer tick.Stop()

	return retry(ctx, f, c.duration, tick, c.options)
}

// Tick implements the Ticker interface.
func (c ConstantTicker) Tick() time.Duration {
	return c.options.Units + c.Jitter()
}
