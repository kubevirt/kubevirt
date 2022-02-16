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
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[Serial][sig-compute]VMIDefaults", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("Disk defaults", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
			// create VMI with missing disk target
			vmi = tests.NewRandomVMI()
			vmi.Spec = v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Disks: []v1.Disk{
							{Name: "testdisk"},
						},
					},
					Resources: v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("8192Ki"),
						},
					},
				},
				Volumes: []v1.Volume{
					{
						Name: "testdisk",
						VolumeSource: v1.VolumeSource{
							ContainerDisk: &v1.ContainerDiskSource{
								Image: "dummy",
							},
						},
					},
				},
			}
		})

		It("[test_id:4115]Should be applied to VMIs", func() {
			// create the VMI first
			_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			newVMI, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// check defaults
			disk := newVMI.Spec.Domain.Devices.Disks[0]
			Expect(disk.Disk).ToNot(BeNil(), "DiskTarget should not be nil")
			Expect(disk.Disk.Bus).ToNot(BeEmpty(), "DiskTarget's bus should not be empty")
		})

	})

	Context("MemBalloon defaults", func() {
		var kvConfiguration v1.KubeVirtConfiguration

		BeforeEach(func() {
			tests.BeforeTestCleanup()
			// create VMI with missing disk target
			vmi = tests.NewRandomVMI()

			kv := util.GetCurrentKv(virtClient)
			kvConfiguration = kv.Spec.Configuration
		})

		AfterEach(func() {
			tests.UpdateKubeVirtConfigValueAndWait(kvConfiguration)
		})

		It("[test_id:4556]Should be present in domain", func() {
			By("Creating a virtual machine")
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for successful start")
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Getting domain of vmi")
			domain, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			expected := api.MemBalloon{
				Model: "virtio-non-transitional",
				Stats: &api.Stats{
					Period: 10,
				},
				Address: &api.Address{
					Type:     "pci",
					Domain:   "0x0000",
					Bus:      "0x04",
					Slot:     "0x00",
					Function: "0x0",
				},
			}
			Expect(domain.Devices.Ballooning).ToNot(BeNil(), "There should be default memballoon device")
			Expect(*domain.Devices.Ballooning).To(Equal(expected), "Default to virtio model and 10 seconds pooling")
		})

		table.DescribeTable("Should override period in domain if present in virt-config ", func(period uint32, expected api.MemBalloon) {
			By("Adding period to virt-config")
			kvConfigurationCopy := kvConfiguration.DeepCopy()
			kvConfigurationCopy.MemBalloonStatsPeriod = &period
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfigurationCopy)

			By("Creating a virtual machine")
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for successful start")
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Getting domain of vmi")
			domain, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Devices.Ballooning).ToNot(BeNil(), "There should be memballoon device")
			Expect(*domain.Devices.Ballooning).To(Equal(expected))
		},
			table.Entry("[test_id:4557]with period 12", uint32(12), api.MemBalloon{
				Model: "virtio-non-transitional",
				Stats: &api.Stats{
					Period: 12,
				},
				Address: &api.Address{
					Type:     "pci",
					Domain:   "0x0000",
					Bus:      "0x04",
					Slot:     "0x00",
					Function: "0x0",
				},
			}),
			table.Entry("[test_id:4558]with period 0", uint32(0), api.MemBalloon{
				Model: "virtio-non-transitional",
				Address: &api.Address{
					Type:     "pci",
					Domain:   "0x0000",
					Bus:      "0x04",
					Slot:     "0x00",
					Function: "0x0",
				},
			}),
		)

		It("[test_id:4559]Should not be present in domain ", func() {
			By("Creating a virtual machine with autoAttachmemballoon set to false")
			f := false
			vmi.Spec.Domain.Devices.AutoattachMemBalloon = &f
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for successful start")
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Getting domain of vmi")
			domain, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			expected := api.MemBalloon{
				Model: "none",
			}
			Expect(domain.Devices.Ballooning).ToNot(BeNil(), "There should be memballoon device")
			Expect(*domain.Devices.Ballooning).To(Equal(expected))
		})

	})

})
