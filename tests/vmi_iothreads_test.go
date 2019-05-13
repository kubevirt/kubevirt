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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"encoding/xml"
	"flag"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("IOThreads", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance
	dedicated := true

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
	})

	Context("IOThreads Policies", func() {

		It("Should honor shared ioThreadsPolicy for single disk", func() {
			policy := v1.IOThreadsPolicyShared
			vmi.Spec.Domain.IOThreadsPolicy = &policy

			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStart(vmi)

			getOptions := metav1.GetOptions{}
			var newVMI *v1.VirtualMachineInstance

			newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &getOptions)
			Expect(err).ToNot(HaveOccurred())

			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())

			expectedIOThreads := 1
			Expect(int(domSpec.IOThreads.IOThreads)).To(Equal(expectedIOThreads))

			Expect(len(newVMI.Spec.Domain.Devices.Disks)).To(Equal(1))
		})

		It("[test_id:864][ref_id:2065] Should honor a mix of shared and dedicated ioThreadsPolicy", func() {
			policy := v1.IOThreadsPolicyShared
			vmi.Spec.Domain.IOThreadsPolicy = &policy

			// The disk that came with the VMI
			vmi.Spec.Domain.Devices.Disks[0].DedicatedIOThread = &dedicated

			tests.AddEphemeralDisk(vmi, "shr1", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			tests.AddEphemeralDisk(vmi, "shr2", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))

			By("Creating VMI with 1 dedicated and 2 shared ioThreadPolicies")
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStart(vmi)

			getOptions := metav1.GetOptions{}
			var newVMI *v1.VirtualMachineInstance

			By("Fetching the VMI from the cluster")
			newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &getOptions)
			Expect(err).ToNot(HaveOccurred())

			By("Fetching the domain XML from the running pod")
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())

			By("Verifying the total number of ioThreads")
			expectedIOThreads := 2
			Expect(int(domSpec.IOThreads.IOThreads)).To(Equal(expectedIOThreads))

			By("Ensuring there are the expected number of disks")
			Expect(len(newVMI.Spec.Domain.Devices.Disks)).To(Equal(len(vmi.Spec.Domain.Devices.Disks)))

			By("Verifying the ioThread mapping for disks")
			disk0, err := getDiskByName(domSpec, "disk0")
			Expect(err).ToNot(HaveOccurred())
			disk1, err := getDiskByName(domSpec, "shr1")
			Expect(err).ToNot(HaveOccurred())
			disk2, err := getDiskByName(domSpec, "shr2")
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the ioThread ID for dedicated disk is unique")
			Expect(*disk1.Driver.IOThread).To(Equal(*disk2.Driver.IOThread))
			By("Ensuring that the ioThread ID's for shared disks are equal")
			Expect(*disk0.Driver.IOThread).ToNot(Equal(*disk1.Driver.IOThread))
		})

		table.DescribeTable("[ref_id:2065] should honor auto ioThreadPolicy", func(numCpus int, expectedIOThreads int) {
			policy := v1.IOThreadsPolicyAuto
			vmi.Spec.Domain.IOThreadsPolicy = &policy

			vmi.Spec.Domain.Devices.Disks[0].DedicatedIOThread = &dedicated

			tests.AddEphemeralDisk(vmi, "ded2", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[1].DedicatedIOThread = &dedicated

			tests.AddEphemeralDisk(vmi, "shr1", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			tests.AddEphemeralDisk(vmi, "shr2", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			tests.AddEphemeralDisk(vmi, "shr3", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			tests.AddEphemeralDisk(vmi, "shr4", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))

			cpuReq := resource.MustParse(fmt.Sprintf("%d", numCpus))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = cpuReq

			By("Creating VMI with 2 dedicated and 4 shared ioThreadPolicies")
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStart(vmi)

			getOptions := metav1.GetOptions{}
			var newVMI *v1.VirtualMachineInstance

			By("Fetching the VMI from the cluster")
			newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &getOptions)
			Expect(err).ToNot(HaveOccurred())

			By("Fetching the domain XML from the running pod")
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())

			By("Verifying the total number of ioThreads")
			Expect(int(domSpec.IOThreads.IOThreads)).To(Equal(expectedIOThreads))

			By("Ensuring there are the expected number of disks")
			Expect(len(newVMI.Spec.Domain.Devices.Disks)).To(Equal(len(vmi.Spec.Domain.Devices.Disks)))

			By("Verifying the ioThread mapping for disks")
			disk0, err := getDiskByName(domSpec, "disk0")
			Expect(err).ToNot(HaveOccurred())
			ded2, err := getDiskByName(domSpec, "ded2")
			Expect(err).ToNot(HaveOccurred())
			shr1, err := getDiskByName(domSpec, "shr1")
			Expect(err).ToNot(HaveOccurred())
			shr2, err := getDiskByName(domSpec, "shr2")
			Expect(err).ToNot(HaveOccurred())
			shr3, err := getDiskByName(domSpec, "shr2")
			Expect(err).ToNot(HaveOccurred())
			shr4, err := getDiskByName(domSpec, "shr2")
			Expect(err).ToNot(HaveOccurred())

			// the ioThreads for disks sh1 through shr4 will vary based on how many CPUs there are
			// but we already verified the total number of threads, so we know they're spread out
			// across the proper threadId pool.

			By("Ensuring disk0 has a unique threadId")
			Expect(*disk0.Driver.IOThread).ToNot(Equal(*ded2.Driver.IOThread), "disk0 should have a dedicated ioThread")
			Expect(*disk0.Driver.IOThread).ToNot(Equal(*shr1.Driver.IOThread), "disk0 should have a dedicated ioThread")
			Expect(*disk0.Driver.IOThread).ToNot(Equal(*shr2.Driver.IOThread), "disk0 should have a dedicated ioThread")
			Expect(*disk0.Driver.IOThread).ToNot(Equal(*shr3.Driver.IOThread), "disk0 should have a dedicated ioThread")
			Expect(*disk0.Driver.IOThread).ToNot(Equal(*shr4.Driver.IOThread), "disk0 should have a dedicated ioThread")

			By("Ensuring ded2 has a unique threadId")
			Expect(*ded2.Driver.IOThread).ToNot(Equal(*shr1.Driver.IOThread), "ded2 should have a dedicated ioThread")
			Expect(*ded2.Driver.IOThread).ToNot(Equal(*shr2.Driver.IOThread), "ded2 should have a dedicated ioThread")
			Expect(*ded2.Driver.IOThread).ToNot(Equal(*shr3.Driver.IOThread), "ded2 should have a dedicated ioThread")
			Expect(*ded2.Driver.IOThread).ToNot(Equal(*shr4.Driver.IOThread), "ded2 should have a dedicated ioThread")
		},
			// special case: there's always at least one thread for the shared pool:
			// two dedicated and one shared thread is 3 threads.
			table.Entry("for one CPU", 1, 3),
			table.Entry("[test_id:856] for two CPUs", 2, 4),
			table.Entry("[test_id:856] for three CPUs", 3, 6),
			// there's only 6 threads expected because there's 6 total disks, even
			// though the limit would have supported 8.
			table.Entry("for four CPUs", 4, 6),
		)
	})
})

func getDiskByName(domSpec *api.DomainSpec, diskName string) (*api.Disk, error) {
	for _, disk := range domSpec.Devices.Disks {
		if disk.Alias.Name == diskName {
			return &disk, nil
		}
	}
	return nil, fmt.Errorf("disk device '%s' not found", diskName)
}
