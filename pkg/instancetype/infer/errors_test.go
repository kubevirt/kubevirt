/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package infer_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/instancetype/infer"
)

var _ = Describe("Instancetype errors", func() {
	Context("IgnoreableInferenceError", func() {
		It("Passes through error message of wrapped error", func() {
			err := errors.New("test error")
			ignoreableInferenceError := infer.NewIgnoreableInferenceError(err)
			Expect(ignoreableInferenceError.Error()).To(Equal(err.Error()))
		})

		It("Passes through wrapped error on unwrap", func() {
			err := errors.New("test error")
			ignoreableInferenceError := infer.NewIgnoreableInferenceError(err)
			Expect(errors.Unwrap(ignoreableInferenceError)).To(Equal(err))
		})
	})
})
