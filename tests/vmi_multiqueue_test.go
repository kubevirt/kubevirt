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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"encoding/xml"
	"fmt"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/testsuite"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
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
			vmi := tests.NewRandomFedoraVMI()
			Expect(numCpus).To(BeNumerically("<=", availableCPUs),
				fmt.Sprintf("Testing environment only has nodes with %d CPUs available, but required are %d CPUs", availableCPUs, numCpus),
			)
			cpuReq := resource.MustParse(fmt.Sprintf("%d", numCpus))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = cpuReq
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.Bool(true)

			vmi.Spec.Domain.Devices.Interfaces[0].Model = interfaceModel

			By("Creating and starting the VMI")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
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

		It("[test_id:959][rfe_id:2065] Should honor multiQueue requests", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			Expect(numCpus).To(BeNumerically("<=", availableCPUs),
				fmt.Sprintf("Testing environment only has nodes with %d CPUs available, but required are %d CPUs", availableCPUs, numCpus),
			)

			vmi.Spec.Domain.Devices.BlockMultiQueue = pointer.Bool(true)
			cpuReq := resource.MustParse(fmt.Sprintf("%d", numCpus))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = cpuReq

			tests.AddEphemeralDisk(vmi, "disk1", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))

			By("Creating VMI with 2 disks, 3 CPUs and multi-queue enabled")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			libwait.WaitForSuccessfulVMIStart(vmi)

			getOptions := metav1.GetOptions{}
			var newVMI *v1.VirtualMachineInstance

			By("Fetching VMI from cluster")
			newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &getOptions)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying VMI")
			newCpuReq := newVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]
			Expect(int32(newCpuReq.Value())).To(Equal(numCpus))
			Expect(*newVMI.Spec.Domain.Devices.BlockMultiQueue).To(BeTrue())

			By("Fetching Domain XML from running pod")
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())

			By("Ensuring each disk has three queues assigned")
			for _, disk := range domSpec.Devices.Disks {
				Expect(int32(*disk.Driver.Queues)).To(Equal(numCpus))
			}
		})

		It("should be able to create a multi-queue VMI when requesting a single vCPU", func() {
			vmi := libvmi.NewCirros()

			vmi.Spec.Domain.CPU = &v1.CPU{Cores: 1, Sockets: 1, Threads: 1}
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.Bool(true)

			By("Creating and starting the VMI")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Fetching Domain XML from running pod")
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())

			for i, iface := range domSpec.Devices.Interfaces {
				expectedIfaceName := fmt.Sprintf("tap%d", i)

				Expect(iface.Target.Device).To(Equal(expectedIfaceName), fmt.Sprintf("the target name should be %s", expectedIfaceName))
				Expect(iface.Target.Managed).To(Equal("no"), "we should instruct libvirt not to configure the tap device")
			}
		})
	})
})
