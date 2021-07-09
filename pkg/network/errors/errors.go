package errors

import "fmt"

type CriticalNetworkError struct {
	Msg string
}

func (e CriticalNetworkError) Error() string { return e.Msg }

func CreateCriticalNetworkError(err error) *CriticalNetworkError {
	return &CriticalNetworkError{Msg: fmt.Sprintf("Critical network error: %v", err)}
}
