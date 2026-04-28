/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package migrationdomainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type possiblyFinishedQueue struct {
	isFinished bool
}

func (q *possiblyFinishedQueue) all() ([]result, bool) {
	return nil, q.isFinished
}

func (*possiblyFinishedQueue) startPolling() {
	panic("not implemented")
}

var _ = Describe("Handler", func() {
	var h *handler

	BeforeEach(func() {
		h = &handler{
			vmiStats: make(map[string]vmiQueue),
		}
	})

	Describe("Collect", func() {
		Context("when queue is finished", func() {
			It("should delete the queue after collecting results", func() {
				h.vmiStats["default/test-vmi"] = &possiblyFinishedQueue{
					isFinished: true,
				}
				h.Collect()

				Expect(h.vmiStats).To(BeEmpty())
			})
		})

		Context("when queue is not finished", func() {
			It("should keep the queue after collecting results", func() {
				h.vmiStats["default/test-vmi"] = &possiblyFinishedQueue{
					isFinished: false,
				}
				h.Collect()

				Expect(h.vmiStats).To(HaveKey("default/test-vmi"))
			})
		})
	})
})
