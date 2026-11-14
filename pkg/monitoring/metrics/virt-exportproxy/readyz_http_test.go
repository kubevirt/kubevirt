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
 * Copyright The KubeVirt Authors.
 */

package virtexportproxy

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/exportproxy/admission"
)

var _ = Describe("WriteReadyzResponse", func() {
	BeforeEach(func() {
		resetTransferMetricsState()
	})

	It("returns 200 when the pod is ready for service", func() {
		recorder := httptest.NewRecorder()
		WriteReadyzResponse(recorder)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(Equal(readyzOKBody))
	})

	It("returns 503 when active transfers reach the hard limit", func() {
		setActiveTransferCountForTest(admission.HardTransferLimit)
		Expect(ReadyForService()).To(BeFalse())

		recorder := httptest.NewRecorder()
		WriteReadyzResponse(recorder)

		Expect(recorder.Code).To(Equal(http.StatusServiceUnavailable))
		Expect(recorder.Body.String()).To(Equal(readyzOverloadedBody))
	})

	It("returns 503 while shedding until active transfers drop to the clear threshold", func() {
		setActiveTransferCountForTest(admission.HardTransferLimit)
		Expect(ReadyForService()).To(BeFalse())

		setActiveTransferCountForTest(admission.HardTransferClear + 1)

		recorder := httptest.NewRecorder()
		WriteReadyzResponse(recorder)

		Expect(recorder.Code).To(Equal(http.StatusServiceUnavailable))
		Expect(recorder.Body.String()).To(Equal(readyzOverloadedBody))
	})

	It("returns 200 again after shedding clears at the hard transfer clear threshold", func() {
		setActiveTransferCountForTest(admission.HardTransferLimit)
		Expect(ReadyForService()).To(BeFalse())

		setActiveTransferCountForTest(admission.HardTransferClear)
		Expect(ReadyForService()).To(BeTrue())

		recorder := httptest.NewRecorder()
		WriteReadyzResponse(recorder)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(Equal(readyzOKBody))
	})
})
