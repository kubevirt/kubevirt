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
