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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/exportproxy/admission"
)

var heldTransfers []activeTransfer

type testUtilizationReader struct {
	cpuPercent float64
	memPercent float64
	ok         bool
}

func (t testUtilizationReader) Utilization() (cpuPercent, memoryPercent float64, ok bool) {
	return t.cpuPercent, t.memPercent, t.ok
}

var _ = Describe("ReadyForService", func() {
	BeforeEach(func() {
		resetTransferMetricsState()
	})

	It("is ready below the hard limit", func() {
		setActiveTransferCountForTest(admission.HardTransferLimit - 1)
		Expect(ReadyForService()).To(BeTrue())
	})

	It("becomes not ready at the hard limit", func() {
		setActiveTransferCountForTest(admission.HardTransferLimit)
		Expect(ReadyForService()).To(BeFalse())
	})

	It("stays not ready until active transfers drop to the clear threshold", func() {
		setActiveTransferCountForTest(admission.HardTransferLimit)
		Expect(ReadyForService()).To(BeFalse())

		setActiveTransferCountForTest(admission.HardTransferClear + 1)
		Expect(ReadyForService()).To(BeFalse())

		setActiveTransferCountForTest(admission.HardTransferClear)
		Expect(ReadyForService()).To(BeTrue())
	})
})

var _ = Describe("TryRecordTransferStarted admission", func() {
	BeforeEach(func() {
		heldTransfers = nil
		resetTransferMetricsState()
	})

	AfterEach(func() {
		for _, transfer := range heldTransfers {
			transfer.Finish()
		}
		heldTransfers = nil
		resetTransferMetricsState()
	})

	It("rejects new transfers at the soft limit", func() {
		for i := int64(0); i < admission.SoftTransferLimit; i++ {
			transfer, ok := TryRecordTransferStarted()
			Expect(ok).To(BeTrue())
			heldTransfers = append(heldTransfers, transfer)
		}

		_, ok := TryRecordTransferStarted()
		Expect(ok).To(BeFalse())
		Expect(ActiveTransferCount()).To(Equal(admission.SoftTransferLimit))
	})

	DescribeTable("rejects new transfers based on soft utilization limits",
		func(reader testUtilizationReader, expectOK bool) {
			admission.SetUtilizationReaderForTest(reader)
			transfer, ok := TryRecordTransferStarted()
			Expect(ok).To(Equal(expectOK))
			if expectOK {
				heldTransfers = append(heldTransfers, transfer)
			} else {
				Expect(ActiveTransferCount()).To(Equal(int64(0)))
			}
		},
		Entry("cpu over threshold", testUtilizationReader{cpuPercent: 71, ok: true}, false),
		Entry("memory over threshold", testUtilizationReader{memPercent: 80, ok: true}, false),
		Entry("both below threshold", testUtilizationReader{cpuPercent: 50, memPercent: 50, ok: true}, true),
		Entry("unavailable metrics fail open", testUtilizationReader{cpuPercent: 100, ok: false}, true),
	)

	It("accepts transfers below the soft limit", func() {
		for i := int64(0); i < admission.SoftTransferLimit-1; i++ {
			transfer, ok := TryRecordTransferStarted()
			Expect(ok).To(BeTrue())
			heldTransfers = append(heldTransfers, transfer)
		}

		transfer, ok := TryRecordTransferStarted()
		Expect(ok).To(BeTrue())
		heldTransfers = append(heldTransfers, transfer)
		Expect(ActiveTransferCount()).To(Equal(admission.SoftTransferLimit))
		Expect(getGaugeValue(activeTransfers)).To(Equal(float64(admission.SoftTransferLimit)))
	})
})
