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

package subresources

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmops"
)

// Only checks for the default which is non configured usbredir.
// The functest for configured usbredir is under tests/virtctl/usbredir.go
var _ = Describe(compute.SIG("usbredir support", func() {

	const enoughMemForSafeBiosEmulation = "32Mi"

	It("should fail to connect to VMI's usbredir socket", func() {
		vmi := libvmi.New(libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation))
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
		virtClient := kubevirt.Client()
		usbredirVMI, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).USBRedir(vmi.ObjectMeta.Name)
		Expect(err).To(HaveOccurred())
		Expect(usbredirVMI).To(BeNil())
	})
}))
