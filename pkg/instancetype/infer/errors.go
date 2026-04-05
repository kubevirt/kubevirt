/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package infer

type IgnoreableInferenceError struct {
	err error
}

func (e *IgnoreableInferenceError) Error() string {
	return e.err.Error()
}

func (e *IgnoreableInferenceError) Unwrap() error {
	return e.err
}

func NewIgnoreableInferenceError(err error) error {
	return &IgnoreableInferenceError{err: err}
}
