package breaker

import (
	"net/http"
	"time"
)

// StatusCodeValidator is a function that determines if a status code written
// to a client by a circuit breaking Handler should count as a success or
// failure. The DefaultStatusCodeValidator can be used in most situations.
type StatusCodeValidator func(int) bool

// Middleware produces an http.Handler factory like Handler to be composed.
func Middleware(breaker Breaker, validator StatusCodeValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Handler(breaker, validator, next)
	}
}

// Handler produces an http.Handler that's governed by the passed Breaker and
// StatusCodeValidator. Responses written by the next http.Handler whose
// status codes fail the validator signal failures to the breaker. Once the
// breaker opens, incoming requests are terminated before being forwarded with
// HTTP 503.
func Handler(breaker Breaker, validator StatusCodeValidator, next http.Handler) http.Handler {
	return &handler{
		breaker:   breaker,
		validator: validator,
		next:      next,
	}
}

type handler struct {
	breaker   Breaker
	validator StatusCodeValidator
	next      http.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.breaker.Allow() {
		h.serveClosed(w, r)
	} else {
		h.serveOpened(w, r)
	}
}

func (h *handler) serveClosed(w http.ResponseWriter, r *http.Request) {
	cw := &codeWriter{w, 200}
	begin := time.Now()

	h.next.ServeHTTP(cw, r)

	duration := time.Since(begin)
	if h.validator(cw.code) {
		h.breaker.Success(duration)
	} else {
		h.breaker.Failure(duration)
	}
}

func (h *handler) serveOpened(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

type codeWriter struct {
	http.ResponseWriter
	code int
}

func (w *codeWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// DefaultStatusCodeValidator considers any status code less than 500 to be a
// success, from the perspective of a server. All other codes are failures.
func DefaultStatusCodeValidator(code int) bool {
	return code < 500
}
