package errors

import "fmt"

type CriticalNetworkError struct {
	wrappedErr error
	Msg        string
}

func (e CriticalNetworkError) Error() string { return e.Msg }
func (e CriticalNetworkError) Unwrap() error { return e.wrappedErr }

func CreateCriticalNetworkError(err error) *CriticalNetworkError {
	return &CriticalNetworkError{
		wrappedErr: err,
		Msg:        fmt.Sprintf("Critical network error: %v", err),
	}
}
