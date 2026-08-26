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

package disk_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/controller/disk"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Disk Controller", func() {
	const (
		preferredBus   = v1.DiskBusVirtio
		preferenceName = "test-preference"
	)

	type diskController interface {
		ApplyDiskPreferences(*v1.VirtualMachine, *v1.VirtualMachineInstanceSpec) error
	}

	var (
		ctrl          diskController
		fakeClientset *fake.Clientset
		virtClient    *kubecli.MockKubevirtClient
	)

	BeforeEach(func() {
		mockCtrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(mockCtrl)
		fakeClientset = fake.NewSimpleClientset()

		virtClient.EXPECT().VirtualMachinePreference(metav1.NamespaceDefault).Return(
			fakeClientset.InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(
			fakeClientset.InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

		preference := &instancetypev1beta1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      preferenceName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				Devices: &instancetypev1beta1.DevicePreferences{
					PreferredDiskBus: preferredBus,
				},
			},
		}
		_, err := fakeClientset.InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault).
			Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		ctrl = disk.New(nil, nil, nil, virtClient)
	})

	DescribeTable("ApplyDiskPreferences does nothing when", func(vm *v1.VirtualMachine) {
		originalSpec := vm.Spec.Template.Spec.DeepCopy()
		Expect(ctrl.ApplyDiskPreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())
		Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(Equal(originalSpec.Domain.Devices.Disks))
	},
		Entry("the VM has no preference reference",
			libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithDisk("disk1", v1.DiskBusVirtio))),
		),
		Entry("the VMI spec has no disks",
			libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace(k8sv1.NamespaceDefault)),
				libvmi.WithPreference(preferenceName)),
		),
	)

	It("should apply the preferred disk bus to disks without a bus", func() {
		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace(k8sv1.NamespaceDefault),
			libvmi.WithDisk("disk1", ""),
			libvmi.WithDisk("disk2", ""),
		), libvmi.WithPreference(preferenceName))

		Expect(ctrl.ApplyDiskPreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())

		for _, d := range vm.Spec.Template.Spec.Domain.Devices.Disks {
			Expect(d.DiskDevice.Disk.Bus).To(Equal(preferredBus), "disk %q should have the preferred bus", d.Name)
		}
	})

	It("should not override a disk bus that is already set", func() {
		const existingBus = v1.DiskBusSCSI
		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace(k8sv1.NamespaceDefault),
			libvmi.WithDisk("disk1", ""),
			libvmi.WithDisk("disk2", ""),
		), libvmi.WithPreference(preferenceName))
		// Set bus after VM creation because WithPreference clears all disk buses
		vm.Spec.Template.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = existingBus

		Expect(ctrl.ApplyDiskPreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())

		Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal(existingBus),
			"existing bus should not be overridden")
		Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferredBus),
			"empty bus should get the preferred bus")
	})

	It("should do nothing when the preference has no device preferences", func() {
		noDevicePreference := &instancetypev1beta1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-device-preference",
				Namespace: metav1.NamespaceDefault,
			},
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{},
		}
		_, err := fakeClientset.InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault).
			Create(context.Background(), noDevicePreference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace(k8sv1.NamespaceDefault),
			libvmi.WithDisk("disk1", ""),
		), libvmi.WithPreference(noDevicePreference.Name))

		Expect(ctrl.ApplyDiskPreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())

		for _, d := range vm.Spec.Template.Spec.Domain.Devices.Disks {
			Expect(d.DiskDevice.Disk.Bus).To(BeEmpty(), "disk %q should have empty bus", d.Name)
		}
	})
})
