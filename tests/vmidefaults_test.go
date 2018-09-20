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
	"flag"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VMIDefaults", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance

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

	Context("Disk defaults", func() {

		It("Should be applied to VMIs", func() {
			// create the VMI first
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			newVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// check defaults
			disk := newVMI.Spec.Domain.Devices.Disks[0]
			Expect(disk.Disk).ToNot(BeNil(), "DiskTarget should not be nil")
			Expect(disk.Disk.Bus).ToNot(BeEmpty(), "DiskTarget's bus should not be empty")
		})

	})

})
