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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("MultiQueue VMI", func() {
	const numCpus uint32 = 3

	DescribeTable("should boot fedora to the login prompt and report the correct number of queues",
		func(interfaceModel string, expectedQueueCount int32) {
			availableCPUs := libnode.GetHighestCPUNumberAmongNodes(kubevirt.Client())
			Expect(numCpus).To(BeNumerically("<=", availableCPUs),
				fmt.Sprintf("Testing environment only has nodes with %d CPUs available, but required are %d CPUs", availableCPUs, numCpus),
			)

			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceWithModel(libvmi.InterfaceDeviceWithMasqueradeBinding(), interfaceModel)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetworkInterfaceMultiQueue(true),
				libvmi.WithCPUCount(numCpus, 1, 1),
			)

			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).
				Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Checking QueueCount has the expected value")
			Expect(vmi.Status.Interfaces[0].QueueCount).To(Equal(expectedQueueCount))
		},
		Entry("[test_id:4599] with default virtio interface", v1.VirtIO, int32(numCpus)),
		Entry("with e1000 interface", "e1000", int32(1)),
	)
}))
