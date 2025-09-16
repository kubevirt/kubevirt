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

package tests_test

import (
	"context"
	"fmt"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/testsuite"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[sig-compute]MultiQueue", decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("MultiQueue Behavior", func() {
		var availableCPUs int
		const numCpus int32 = 3

		BeforeEach(func() {
			availableCPUs = libnode.GetHighestCPUNumberAmongNodes(virtClient)
		})

		DescribeTable("should be able to successfully boot fedora to the login prompt with multi-queue without being blocked by selinux", func(interfaceModel string, expectedQueueCount int32) {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			Expect(numCpus).To(BeNumerically("<=", availableCPUs),
				fmt.Sprintf("Testing environment only has nodes with %d CPUs available, but required are %d CPUs", availableCPUs, numCpus),
			)
			cpuReq := resource.MustParse(fmt.Sprintf("%d", numCpus))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = cpuReq
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.P(true)

			vmi.Spec.Domain.Devices.Interfaces[0].Model = interfaceModel

			By("Creating and starting the VMI")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Checking if we can login")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Checking QueueCount has the expected value")
			Expect(vmi.Status.Interfaces[0].QueueCount).To(Equal(expectedQueueCount))
		},
			Entry("[test_id:4599] with default virtio interface", v1.VirtIO, numCpus),
			Entry("with e1000 interface", "e1000", int32(1)),
		)
	})
})
