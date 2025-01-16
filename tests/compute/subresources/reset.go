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
 * Copyright The KubeVirt Authors
 *
 */

package subresources

import (
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = compute.SIGDescribe("Reset subresource", func() {

	Describe("Reset a VirtualMachineInstance", func() {
		const vmiLaunchTimeout = 360

		It("should succeed", func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpineWithTestTooling(), vmiLaunchTimeout)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			errChan := make(chan error)
			go func() {
				time.Sleep(5)
				errChan <- kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Reset(context.Background(), vmi.Name)
			}()

			start := time.Now().UTC().Unix()

			By(fmt.Sprintf("Waiting for vmi %s reset", vmi.Name))
			if vmi.Namespace == "" {
				vmi.Namespace = testsuite.GetTestNamespace(vmi)
			}
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ".*Hypervisor detected.*KVM.*"},
			}, 300)

			end := time.Now().UTC().Unix()

			if err != nil {
				err = fmt.Errorf("start [%d] end [%d] err: %v", start, end, err)
			}
			Expect(err).ToNot(HaveOccurred())

			select {
			case err := <-errChan:
				Expect(err).ToNot(HaveOccurred())
			}
		})
	})
})
