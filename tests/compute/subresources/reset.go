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

package subresources

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(compute.SIG("Reset subresource", func() {

	Describe("Reset a VirtualMachineInstance", func() {
		const vmiLaunchTimeout = 360

		It("should succeed", func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpineWithTestTooling(), vmiLaunchTimeout)
			oldUID := vmi.UID

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Store boot time pre and post reset")
			cmd := "cat /proc/stat | grep btime"
			bTimePreReset, err := console.RunCommandAndStoreOutput(vmi, cmd, time.Second*30)
			Expect(err).ToNot(HaveOccurred())

			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Reset(context.Background(), vmi.Name)
			Expect(err).ToNot(HaveOccurred())

			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			bTimePostReset, err := console.RunCommandAndStoreOutput(vmi, cmd, time.Second*30)
			Expect(err).ToNot(HaveOccurred())

			By("Check the pre and post reset boot times are different and non-empty")
			Expect(bTimePreReset).ToNot(BeEmpty())
			Expect(bTimePostReset).ToNot(BeEmpty())
			Expect(bTimePreReset).ToNot(Equal(bTimePostReset))

			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.UID).To(Equal(oldUID))
		})
	})
}))
