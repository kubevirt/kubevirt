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
 *
 */

package virt_controller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
)

var _ = Describe("Memory Metrics", func() {
	Context("kubevirt_vmi_launcher_memory_overhead_bytes", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-namespace",
				Name:      "test-name",
			},
		}

		It("should set the memory overhead correctly as bytes", func() {
			overhead := *resource.NewScaledQuantity(0, resource.Kilo)
			overhead.Add(resource.MustParse("8Mi"))

			metrics.SetVmiLaucherMemoryOverhead(vmi, overhead)

			value, err := metrics.GetVmiLaucherMemoryOverhead(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(float64(8 * 1024 * 1024)))
		})
	})
})
