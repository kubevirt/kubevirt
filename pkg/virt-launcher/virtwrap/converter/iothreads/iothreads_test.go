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

package iothreads

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("GetIOThreadsCountType", func() {
	It("should not panic when IOThreadsPolicy is supplementalPool but IOThreads is nil", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					IOThreadsPolicy: new(v1.IOThreadsPolicySupplementalPool),
					Devices: v1.Devices{
						Disks: []v1.Disk{
							{Name: "disk0", DedicatedIOThread: new(true)},
						},
					},
				},
			},
		}

		Expect(func() {
			GetIOThreadsCountType(vmi)
		}).NotTo(Panic())

		ioThreadCount, autoThreads := GetIOThreadsCountType(vmi)
		Expect(ioThreadCount).To(Equal(2))
		Expect(autoThreads).To(Equal(1))
	})
})
