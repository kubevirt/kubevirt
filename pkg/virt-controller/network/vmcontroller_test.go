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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2024 Red Hat, Inc.
 *
 */

package network_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-controller/network"
)

var _ = Describe("VM Network Controller", func() {
	It("sync does nothing when the hotplug FG is unset", func() {
		c := network.NewVMNetController(fake.NewSimpleClientset(), stubClusterConfig{}, stubPodGetter{})
		Expect(c.Sync(newEmptyVM(), libvmi.New())).To(Equal(newEmptyVM()))
	})

	DescribeTable("sync does nothing when", func(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, podGetter stubPodGetter) {
		c := network.NewVMNetController(fake.NewSimpleClientset(), stubClusterConfig{netHotplugEnabled: true}, podGetter)
		originalVM := vm.DeepCopy()
		Expect(c.Sync(vm, vmi)).To(Equal(originalVM))
	},
		Entry("the VM is not running (there is no VMI)", newEmptyVM(), nil, stubPodGetter{}),
		Entry(
			"the VMI is marked for deletion",
			newEmptyVM(),
			&v1.VirtualMachineInstance{ObjectMeta: k8smetav1.ObjectMeta{DeletionTimestamp: &k8smetav1.Time{}}},
			stubPodGetter{},
		),
		Entry("the VM & VMI have no interfaces and no pod is found", newEmptyVM(), libvmi.New(), stubPodGetter{}),
		Entry(
			"the VM & VMI have no interfaces",
			newEmptyVM(),
			libvmi.New(),
			stubPodGetter{pod: &k8sv1.Pod{}},
		),
		Entry(
			"the VM & VMI have identical interfaces",
			libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)),
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			stubPodGetter{pod: &k8sv1.Pod{}},
		),
	)

	It("sync fails when pod fetching returns an error", func() {
		c := network.NewVMNetController(
			fake.NewSimpleClientset(),
			stubClusterConfig{netHotplugEnabled: true},
			stubPodGetter{err: errors.New("test")},
		)
		updatedVM, err := c.Sync(newEmptyVM(), libvmi.New())
		Expect(err).To(MatchError(isSyncErrorType, "syncError"))
		Expect(updatedVM).To(Equal(newEmptyVM()))
	})

	It("sync fails when VMI patch returns an error", func() {
		clientset := fake.NewSimpleClientset()
		c := network.NewVMNetController(
			clientset,
			stubClusterConfig{netHotplugEnabled: true},
			stubPodGetter{pod: &k8sv1.Pod{}},
		)

		// Setup `Patch` to fail.
		injectedPatchError := errors.New("test patch error")
		clientset.Fake.PrependReactor("patch", "virtualmachineinstances", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			return true, nil, injectedPatchError
		})

		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		vm = plugNetworkInterface(vm, "foonet")

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).To(MatchError(isSyncErrorType, "syncError"))
		Expect(err).To(MatchError(ContainSubstring(injectedPatchError.Error())))
		Expect(updatedVM).To(Equal(originalVM))
	})

	It("sync succeeds to hotplug new interface", func() {
		clientset := fake.NewSimpleClientset()
		c := network.NewVMNetController(
			clientset,
			stubClusterConfig{netHotplugEnabled: true},
			stubPodGetter{pod: &k8sv1.Pod{}},
		)
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vm := libvmi.NewVirtualMachine(vmi.DeepCopy())

		vm = plugNetworkInterface(vm, "foonet")

		// Simulate the existence of the VMI on the server (to allow the Sync to patch it).
		_, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, vmi)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVM).To(Equal(originalVM))

		// Assert that the hotplug reached the VMI
		updatedVMI, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(Equal(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces))
	})

	It("sync succeeds to clear hotunplug interfaces", func() {
		clientset := fake.NewSimpleClientset()
		c := network.NewVMNetController(
			clientset,
			stubClusterConfig{netHotplugEnabled: true},
			stubPodGetter{pod: &k8sv1.Pod{}},
		)
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
		updatedVMI, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		Expect(updatedVMI.Spec.Domain.Devices.Interfaces).To(Equal(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces))
	})

	It("sync does not hotunplug interfaces when pod is not found", func() {
		clientset := fake.NewSimpleClientset()
		c := network.NewVMNetController(
			clientset,
			stubClusterConfig{netHotplugEnabled: true},
			stubPodGetter{pod: nil},
		)
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("foonet")),
			libvmi.WithNetwork(libvmi.MultusNetwork("foonet", "foonet-nad")),
		)
		// Make sure the interfaces are visible on the status as well (otherwise hotunplug is not triggered)
		for _, net := range vmi.Spec.Networks {
			vmi.Status.Interfaces = append(vmi.Status.Interfaces, v1.VirtualMachineInstanceNetworkInterface{Name: net.Name})
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
		updatedVMI, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		iface := vmispec.LookupInterfaceByName(updatedVMI.Spec.Domain.Devices.Interfaces, unplugNetworkName)
		Expect(iface).NotTo(BeNil())
		Expect(iface.State).NotTo(Equal(v1.InterfaceStateAbsent))

	})

	It("sync does not hotunplug interfaces when legacy ordinal interface names are found", func() {
		clientset := fake.NewSimpleClientset()
		pod := &k8sv1.Pod{
			ObjectMeta: k8smetav1.ObjectMeta{Annotations: map[string]string{
				networkv1.NetworkStatusAnnot: `[
						{"interface":"eth0", "name":"default", "namespace": "default"},
						{"interface":"net1", "name":"foonet-nad", "namespace": "default"}
					]`,
			}},
		}
		c := network.NewVMNetController(
			clientset,
			stubClusterConfig{netHotplugEnabled: true},
			stubPodGetter{pod: pod},
		)
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("foonet")),
			libvmi.WithNetwork(libvmi.MultusNetwork("foonet", "foonet-nad")),
		)
		// Make sure the interfaces are visible on the status as well (otherwise hotunplug is not triggered)
		for _, net := range vmi.Spec.Networks {
			vmi.Status.Interfaces = append(vmi.Status.Interfaces, v1.VirtualMachineInstanceNetworkInterface{Name: net.Name})
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

		// Assert that the hotunplug did **not** reached the VMI
		updatedVMI, err := clientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedVMI.Spec.Networks).To(Equal(updatedVM.Spec.Template.Spec.Networks))
		iface := vmispec.LookupInterfaceByName(updatedVMI.Spec.Domain.Devices.Interfaces, unplugNetworkName)
		Expect(iface).NotTo(BeNil())
		Expect(iface.State).NotTo(Equal(v1.InterfaceStateAbsent))
	})
})

type stubClusterConfig struct {
	netHotplugEnabled bool
}

func (s stubClusterConfig) HotplugNetworkInterfacesEnabled() bool {
	return s.netHotplugEnabled
}

type stubPodGetter struct {
	pod *k8sv1.Pod
	err error
}

func (s stubPodGetter) CurrentPod(_ *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	return s.pod, s.err
}

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

func plugNetworkInterface(vm *v1.VirtualMachine, netName string) *v1.VirtualMachine {
	vm.Spec.Template.Spec.Domain.Devices.Interfaces = append(
		vm.Spec.Template.Spec.Domain.Devices.Interfaces,
		libvmi.InterfaceDeviceWithBridgeBinding(netName),
	)
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
