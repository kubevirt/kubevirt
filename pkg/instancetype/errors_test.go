/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package instancetype_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/instancetype"
)

var _ = Describe("Instancetype errors", func() {
	Context("IgnoreableInferenceError", func() {
		It("Passes through error message of wrapped error", func() {
			err := errors.New("test error")
			ignoreableInferenceError := instancetype.NewIgnoreableInferenceError(err)
			Expect(ignoreableInferenceError.Error()).To(Equal(err.Error()))
		})

		It("Passes through wrapped error on unwrap", func() {
			err := errors.New("test error")
			ignoreableInferenceError := instancetype.NewIgnoreableInferenceError(err)
			Expect(errors.Unwrap(ignoreableInferenceError)).To(Equal(err))
		})
	})
})
