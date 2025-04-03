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

package controllers_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	"kubevirt.io/client-go/kubevirt/fake"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/controllers"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("VM Network Controller", func() {
	const (
		defaultNetName   = "default"
		secondaryNetName = "foonet"
		nadName          = "foonet-nad"
	)
	DescribeTable("sync does nothing when", func(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
		c := controllers.NewVMController(fake.NewSimpleClientset())
		originalVM := vm.DeepCopy()
		Expect(c.Sync(vm, vmi)).To(Equal(originalVM))
	},
		Entry("the VM is not running (there is no VMI)", newEmptyVM(), nil),
		Entry(
			"the VMI is marked for deletion",
			newEmptyVM(),
			&v1.VirtualMachineInstance{ObjectMeta: k8smetav1.ObjectMeta{DeletionTimestamp: &k8smetav1.Time{}}},
		),
		Entry("the VM & VMI have no interfaces", newEmptyVM(), libvmi.New()),
		Entry(
			"the VM & VMI have identical interfaces",
			libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, nadName)),
			)),
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, nadName)),
			),
		),
	)

	It("sync fails when VMI patch returns an error", func() {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)

		// Setup `Patch` to fail.
		injectedPatchError := errors.New("test patch error")
		clientset.Fake.PrependReactor(
			"patch",
			"virtualmachineinstances",
			func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, injectedPatchError
			})

		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		vm = plugNetworkInterface(vm, libvmi.InterfaceDeviceWithBridgeBinding("foonet"))

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).To(MatchError(isSyncErrorType, "syncError"))
		Expect(err).To(MatchError(ContainSubstring(injectedPatchError.Error())))
		Expect(updatedVM).To(Equal(originalVM))
	})

	DescribeTable("sync succeeds to hotplug new interface", func(ifaceToPlug v1.Interface) {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		vm = plugNetworkInterface(vm, ifaceToPlug)

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVM).To(Equal(originalVM))

		// Assert that the hotplug reached the VMI
		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(Equal(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces))
	},
		Entry("when the plugged interface uses bridge binding", libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)),
		Entry("when the plugged interface uses SR-IOV binding", libvmi.InterfaceDeviceWithSRIOVBinding(secondaryNetName)),
		Entry("when the plugged interface has link state down", v1.Interface{
			Name: secondaryNetName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				Bridge: &v1.InterfaceBridge{},
			},
			State: v1.InterfaceStateLinkDown,
		}),
		Entry("when the plugged interface has link state up", v1.Interface{
			Name: secondaryNetName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				Bridge: &v1.InterfaceBridge{},
			},
			State: v1.InterfaceStateLinkUp,
		}),
	)

	It("sync does not hotplug a new absent interface", func() {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		originalVMI := vmi.DeepCopy()
		vm := libvmi.NewVirtualMachine(originalVMI)

		absentIfaceToPlug := v1.Interface{
			Name:                   "absentIface",
			State:                  v1.InterfaceStateAbsent,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
		}

		vm = plugNetworkInterface(vm, absentIfaceToPlug)

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		// Assert that the hotplugged interface and its matching network were cleared
		Expect(updatedVM.Spec.Template.Spec.Networks).To(Equal(originalVMI.Spec.Networks))
		Expect(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces).To(Equal(originalVMI.Spec.Domain.Devices.Interfaces))

		// Assert that the hotplug haven't reached the VMI
		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(originalVMI.Spec.Networks))
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(Equal(originalVMI.Spec.Domain.Devices.Interfaces))
	})

	DescribeTable("sync succeeds to mark an existing interface for hotunplug", func(currentIfaceState v1.InterfaceState) {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)

		multusAndDomainInfoSource := vmispec.NewInfoSource(vmispec.InfoSourceMultusStatus, vmispec.InfoSourceDomain)

		secondaryIface := libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)
		secondaryIface.State = currentIfaceState

		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithInterface(secondaryIface),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, nadName)),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
						Name:             defaultNetName,
						PodInterfaceName: namescheme.PrimaryPodInterfaceName,
						InfoSource:       vmispec.InfoSourceDomain,
					}),
					libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
						Name:             secondaryNetName,
						PodInterfaceName: namescheme.GenerateHashedInterfaceName(secondaryNetName),
						InfoSource:       multusAndDomainInfoSource,
					}),
				),
			),
		)
		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		// Mark the secondary interface for hotunplug
		vm.Spec.Template.Spec.Domain.Devices.Interfaces[1].State = v1.InterfaceStateAbsent

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVM).To(Equal(originalVM))

		// Assert that the hotunplug had reached the VMI
		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(Equal(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces))
	},
		Entry("when the current iface state is empty", v1.InterfaceState("")),
		Entry("when the current iface state is absent", v1.InterfaceStateAbsent),
	)

	It("sync does not hotplug a new interface when it uses binding other than bridge or SR-IOV", func() {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)

		vmi := libvmi.New()
		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		plugNetworkInterface(vm, libvmi.InterfaceWithBindingPlugin(secondaryNetName, v1.PluginBinding{Name: "someplugin"}))

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVM).To(Equal(originalVM))

		// Assert that the hotunplug had not reached the VMI
		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(BeEmpty())
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(BeEmpty())
	})

	It("sync succeeds to clear hotunplug interfaces", func() {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)
		unpluggedIface := libvmi.InterfaceDeviceWithBridgeBinding("foonet")
		unpluggedIface.State = v1.InterfaceStateAbsent
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(unpluggedIface),
			libvmi.WithNetwork(libvmi.MultusNetwork("foonet", "foonet-nad")),
		)
		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		// Expect the original VM to have been mutated, removing the unplugged network interfaces
		originalVM.Spec.Template.Spec.Networks = originalVM.Spec.Template.Spec.Networks[:1]
		originalVM.Spec.Template.Spec.Domain.Devices.Interfaces = originalVM.Spec.Template.Spec.Domain.Devices.Interfaces[:1]
		Expect(updatedVM).To(Equal(originalVM))

		// Assert that the hotplug reached the VMI
		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(Equal(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces))
	})

	It("sync does not hotunplug interfaces when nameing scheme is unknown", func() {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("foonet")),
			libvmi.WithNetwork(libvmi.MultusNetwork("foonet", "foonet-nad")),
		)
		// Make sure the interfaces are visible on the status as well (otherwise hotunplug is not triggered)
		for _, net := range vmi.Spec.Networks {
			vmi.Status.Interfaces = append(vmi.Status.Interfaces,
				v1.VirtualMachineInstanceNetworkInterface{Name: net.Name, PodInterfaceName: ""},
			)
		}

		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		// Unplug the network interface at the VM (only).
		const unplugNetworkName = "foonet"
		vm = unplugNetworkInterface(vm, unplugNetworkName)

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVM).To(Equal(originalVM))

		// Assert that the hotunplug did **not** reach the VMI
		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		iface := vmispec.LookupInterfaceByName(updatedVMI.Spec.Domain.Devices.Interfaces, unplugNetworkName)
		Expect(iface).NotTo(BeNil())
		Expect(iface.State).NotTo(Equal(v1.InterfaceStateAbsent))
	})

	DescribeTable("sync updates link state of an existing interface", func(fromState, toState v1.InterfaceState) {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)
		const defaultNetName = "default"
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:  defaultNetName,
				State: fromState,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmistatus.WithStatus(
				libvmistatus.New(libvmistatus.WithInterfaceStatus(
					v1.VirtualMachineInstanceNetworkInterface{Name: defaultNetName},
				)),
			),
		)

		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		vm.Spec.Template.Spec.Domain.Devices.Interfaces[0].State = toState

		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(
			Equal(vm.Spec.Template.Spec.Domain.Devices.Interfaces))

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
	},
		Entry("up to up", v1.InterfaceStateLinkUp, v1.InterfaceStateLinkUp),
		Entry("up to down", v1.InterfaceStateLinkUp, v1.InterfaceStateLinkDown),
		Entry("up to absent", v1.InterfaceStateLinkUp, v1.InterfaceStateAbsent),
		Entry("up to empty", v1.InterfaceStateLinkUp, v1.InterfaceState("")),
		Entry("down to up", v1.InterfaceStateLinkDown, v1.InterfaceStateLinkUp),
		Entry("down to down", v1.InterfaceStateLinkDown, v1.InterfaceStateLinkDown),
		Entry("down to absent", v1.InterfaceStateLinkDown, v1.InterfaceStateAbsent),
		Entry("down to empty", v1.InterfaceStateLinkDown, v1.InterfaceState("")),
		Entry("empty to up", v1.InterfaceState(""), v1.InterfaceStateLinkUp),
		Entry("empty to down", v1.InterfaceState(""), v1.InterfaceStateLinkDown),
		Entry("empty to absent", v1.InterfaceState(""), v1.InterfaceStateAbsent),
		Entry("empty to empty", v1.InterfaceState(""), v1.InterfaceState("")),
	)

	DescribeTable("sync doesn't update link state if hot-unplug is underway ", func(toState v1.InterfaceState) {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)
		const defaultNetName = "default"
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:  defaultNetName,
				State: v1.InterfaceStateAbsent,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmistatus.WithStatus(
				libvmistatus.New(libvmistatus.WithInterfaceStatus(
					v1.VirtualMachineInstanceNetworkInterface{Name: defaultNetName},
				)),
			),
		)

		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		vm.Spec.Template.Spec.Domain.Devices.Interfaces[0].State = toState

		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(vmi.Spec.Networks))
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(Equal(vmi.Spec.Domain.Devices.Interfaces))

		Expect(updatedVM.Spec.Template.Spec.Networks).To(Equal(vm.Spec.Template.Spec.Networks))
		Expect(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces).To(Equal(vm.Spec.Template.Spec.Domain.Devices.Interfaces))
	},
		Entry("absent to up", v1.InterfaceStateLinkUp),
		Entry("absent to down", v1.InterfaceStateLinkDown),
		Entry("absent to empty", v1.InterfaceState("")),
	)

	It("sync does not hotunplug interfaces when legacy ordinal interface names are found", func() {
		clientset := fake.NewSimpleClientset()
		c := controllers.NewVMController(clientset)
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("foonet")),
			libvmi.WithNetwork(libvmi.MultusNetwork("foonet", "foonet-nad")),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithInterfaceStatus(
						v1.VirtualMachineInstanceNetworkInterface{Name: defaultNetName, PodInterfaceName: namescheme.PrimaryPodInterfaceName},
					),
					libvmistatus.WithInterfaceStatus(
						v1.VirtualMachineInstanceNetworkInterface{Name: "foonet", PodInterfaceName: "net1"},
					),
				),
			),
		)

		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		// Unplug the network interface at the VM (only).
		const unplugNetworkName = "foonet"
		vm = unplugNetworkInterface(vm, unplugNetworkName)

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVM).To(Equal(originalVM))

		// Assert that the hotunplug did **not** reached the VMI
		updatedVMI, err := clientset.KubevirtV1().
			VirtualMachineInstances(vmi.Namespace).
			Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		iface := vmispec.LookupInterfaceByName(updatedVMI.Spec.Domain.Devices.Interfaces, unplugNetworkName)
		Expect(iface).NotTo(BeNil())
		Expect(iface.State).NotTo(Equal(v1.InterfaceStateAbsent))
	})
})

type syncError interface {
	error
	Reason() string
	// RequiresRequeue indicates if the sync error should trigger a requeue, or
	// if information should just be added to the sync condition and a regular controller
	// wakeup will resolve the situation.
	RequiresRequeue() bool
}

func isSyncErrorType(e error) bool {
	var errWithReason syncError
	return errors.As(e, &errWithReason)
}

func plugNetworkInterface(vm *v1.VirtualMachine, ifaceToPlug v1.Interface) *v1.VirtualMachine {
	vm.Spec.Template.Spec.Domain.Devices.Interfaces = append(
		vm.Spec.Template.Spec.Domain.Devices.Interfaces,
		ifaceToPlug,
	)

	netName := ifaceToPlug.Name
	vm.Spec.Template.Spec.Networks = append(
		vm.Spec.Template.Spec.Networks,
		*libvmi.MultusNetwork(netName, netName+"-nad"),
	)
	return vm
}

func unplugNetworkInterface(vm *v1.VirtualMachine, netName string) *v1.VirtualMachine {
	iface := vmispec.LookupInterfaceByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces, netName)
	if iface != nil {
		iface.State = v1.InterfaceStateAbsent
	}
	return vm
}

func newEmptyVM() *v1.VirtualMachine {
	return &v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Template: &v1.VirtualMachineInstanceTemplateSpec{}}}
}
