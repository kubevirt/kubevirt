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

package network_test

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

	"kubevirt.io/kubevirt/pkg/instancetype/controller/network"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Network Controller", func() {
	const (
		preferredModel = "virtio"
		preferenceName = "test-preference"
	)

	type networkController interface {
		ApplyInterfacePreferences(*v1.VirtualMachine, *v1.VirtualMachineInstanceSpec) error
	}

	var (
		ctrl          networkController
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
					PreferredInterfaceModel: preferredModel,
				},
			},
		}
		_, err := fakeClientset.InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault).
			Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		ctrl = network.New(nil, nil, nil, virtClient)
	})

	DescribeTable("ApplyInterfacePreferences does nothing when", func(vm *v1.VirtualMachine) {
		originalVMSpec := vm.Spec.Template.Spec.DeepCopy()
		Expect(ctrl.ApplyInterfacePreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())
		Expect(&vm.Spec.Template.Spec).To(Equal(originalVMSpec))
	},
		Entry("the VM has no preference reference",
			libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding())),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)),
		),
		Entry("the VM has no interfaces",
			libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace(k8sv1.NamespaceDefault),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			), libvmi.WithPreference(preferenceName)),
		),
	)

	It("should apply the preferred interface model to interfaces without a model", func() {
		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace(k8sv1.NamespaceDefault),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding())),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.NewInterface("secondary", libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(libvmi.MultusNetwork("secondary", "some-nad"))),
			libvmi.WithPreference(preferenceName),
		)

		Expect(ctrl.ApplyInterfacePreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())

		for _, iface := range vm.Spec.Template.Spec.Domain.Devices.Interfaces {
			Expect(iface.Model).To(Equal(preferredModel), "interface %q should have the preferred model", iface.Name)
		}
	})

	It("should not override an interface model that is already set", func() {
		const existingModel = "e1000e"
		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace(k8sv1.NamespaceDefault),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithModel(existingModel))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.NewInterface("secondary", libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(libvmi.MultusNetwork("secondary", "some-nad")),
		), libvmi.WithPreference(preferenceName))

		Expect(ctrl.ApplyInterfacePreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())

		Expect(vm.Spec.Template.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(existingModel),
			"existing model should not be overridden")
		Expect(vm.Spec.Template.Spec.Domain.Devices.Interfaces[1].Model).To(Equal(preferredModel),
			"empty model should get the preferred model")
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
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding())),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		), libvmi.WithPreference(noDevicePreference.Name))

		originalVM := vm.DeepCopy()
		Expect(ctrl.ApplyInterfacePreferences(vm, &vm.Spec.Template.Spec)).To(Succeed())
		Expect(vm.Spec.Template.Spec.Domain.Devices.Interfaces).To(Equal(originalVM.Spec.Template.Spec.Domain.Devices.Interfaces))
	})
})
