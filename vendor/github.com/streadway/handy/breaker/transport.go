package breaker

import (
	"errors"
	"net/http"
	"time"
)

var (
	// ErrCircuitOpen is returned by the transport when the downstream is
	// unavailable due to a broken circuit.
	ErrCircuitOpen = errors.New("circuit open")
)

// ResponseValidator is a function that determines if an http.Response
// received by a circuit breaking Transport should count as a success or a
// failure. The DefaultResponseValidator can be used in most situations.
type ResponseValidator func(*http.Response) bool

// Transport produces an http.RoundTripper that's governed by the passed
// Breaker and ResponseValidator. Responses that fail the validator signal
// failures to the breaker. Once the breaker opens, outgoing requests are
// terminated before being forwarded with ErrCircuitOpen.
func Transport(breaker Breaker, validator ResponseValidator, next http.RoundTripper) http.RoundTripper {
	return &transport{
		breaker:   breaker,
		validator: validator,
		next:      next,
	}
}

type transport struct {
	breaker   Breaker
	validator ResponseValidator
	next      http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.breaker.Allow() {
		return nil, ErrCircuitOpen
	}

	begin := time.Now()
	resp, err := t.next.RoundTrip(req)

	duration := time.Since(begin)
	if err != nil || !t.validator(resp) {
		t.breaker.Failure(duration)
	} else {
		t.breaker.Success(duration)
	}

	return resp, err
}

// DefaultResponseValidator considers any status code less than 400 to be a
// success, from the perspective of a client. All other codes are failures.
func DefaultResponseValidator(resp *http.Response) bool {
	return resp.StatusCode < 400
}
