package util

import "fmt"

// wrap an error if we want to actually stop processing by the error.
// unwrapping it will return the original error.
// unwrapping a regular error will return nil
func NewProcessingError(err error) error {
	return fmt.Errorf("%w", err)
}
