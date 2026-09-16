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

package tpm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("HasPersistentDevice", func() {
	DescribeTable("should report whether the TPM device is persistent",
		func(tpm *v1.TPMDevice, vmState *v1.VirtualMachineStateSpec, expected bool) {
			vmiSpec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						TPM: tpm,
					},
				},
				VirtualMachineState: vmState,
			}
			Expect(HasPersistentDevice(vmiSpec)).To(Equal(expected))
		},
		Entry("no TPM device", nil, nil, false),
		Entry("device without persistent and no VirtualMachineState", &v1.TPMDevice{}, nil, false),
		Entry("device without persistent and VirtualMachineState", &v1.TPMDevice{}, &v1.VirtualMachineStateSpec{}, true),
		Entry("device with persistent false and VirtualMachineState", &v1.TPMDevice{Persistent: pointer.P(false)}, &v1.VirtualMachineStateSpec{}, false),
		Entry("device with persistent true and no VirtualMachineState", &v1.TPMDevice{Persistent: pointer.P(true)}, nil, true),
	)
})
