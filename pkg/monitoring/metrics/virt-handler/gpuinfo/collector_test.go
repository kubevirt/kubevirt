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

package gpuinfo

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GPU Info Collector", func() {
	Context("collectCallback", func() {
		It("should return nil when cache is not initialized", func() {
			gpuCache = nil
			results := collectCallback()
			Expect(results).To(BeNil())
		})

		It("should return correct collector results from cache", func() {
			gpuCache = &gpuInfoCache{
				nodeName:    "worker-1",
				lastRefresh: time.Now(),
				allocations: []GPUAllocation{
					{
						Namespace: "test-ns",
						PodName:   "virt-launcher-test-vmi-abc123",
						Resource:  "nvidia.com/NVIDIA_A2-2Q",
						UUID:      "gpu-uuid-123",
					},
					{
						Namespace: "test-ns",
						PodName:   "virt-launcher-test-vmi-abc123",
						Resource:  "nvidia.com/NVIDIA_A2-2Q",
						UUID:      "gpu-uuid-456",
					},
				},
			}

			results := collectCallback()
			Expect(results).To(HaveLen(2))

			Expect(results[0].Value).To(Equal(float64(1)))
			Expect(results[0].ConstLabels["node"]).To(Equal("worker-1"))
			Expect(results[0].ConstLabels["namespace"]).To(Equal("test-ns"))
			Expect(results[0].ConstLabels["pod"]).To(Equal("virt-launcher-test-vmi-abc123"))
			Expect(results[0].ConstLabels["resource"]).To(Equal("nvidia.com/NVIDIA_A2-2Q"))
			Expect(results[0].ConstLabels["uuid"]).To(Equal("gpu-uuid-123"))

			Expect(results[1].ConstLabels["uuid"]).To(Equal("gpu-uuid-456"))
		})

		It("should return empty results when cache has no allocations", func() {
			gpuCache = &gpuInfoCache{
				nodeName:    "worker-1",
				lastRefresh: time.Now(),
			}

			results := collectCallback()
			Expect(results).To(BeEmpty())
		})
	})
})
