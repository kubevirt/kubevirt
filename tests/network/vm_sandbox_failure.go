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

package network

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VM with an unresolvable network attachment definition", decorators.Multus, func() {
	It("should surface the pod sandbox creation failure on the VM status", func() {
		testNamespace := testsuite.GetTestNamespace(nil)

		const (
			secondaryNetName = "secondary"
			nonExistentNAD   = "non-existent-nad"
		)

		// A secondary interface attached to a NAD that does not exist makes the CNI
		// fail to set up the pod sandbox, which should be surfaced on the VM.
		vmi := libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.NewInterface(secondaryNetName, libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, nonExistentNAD)),
		)

		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("asserting the VM reports the pod sandbox creation error")
		Eventually(matcher.ThisVM(vm)).
			WithTimeout(2 * time.Minute).
			WithPolling(5 * time.Second).
			Should(matcher.HavePrintableStatus(v1.VirtualMachineStatusErrorPodSandbox))
	})
}))
