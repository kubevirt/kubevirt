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
 *
 */

package passt_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/passt"
)

var _ = Describe("Passt Repair Handler", func() {
	clusterConfigPasst := stubClusterConfig{
		registeredPlugins: map[string]v1.InterfaceBindingPlugin{
			"passt": {
				SidecarImage:                "passt:latest",
				NetworkAttachmentDefinition: "default/passt-network",
			},
		},
	}

	DescribeTable("Should not run passt repair", func(vmi *v1.VirtualMachineInstance) {
		clusterConfigMultiPlugin := stubClusterConfig{
			registeredPlugins: map[string]v1.InterfaceBindingPlugin{
				"passt": {
					SidecarImage:                "passt:latest",
					NetworkAttachmentDefinition: "default/passt-network",
				},
				"tap": {
					DomainAttachmentType: v1.Tap,
				},
				"managedTap": {
					DomainAttachmentType: v1.ManagedTap,
				},
			},
		}

		passtRepairCalled := false
		fakeCommandWithCallCounter := func(s string, instance *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
			passtRepairCalled = true
		}

		handler := passt.NewRepairManagerWithOptions(
			clusterConfigMultiPlugin,
			stubFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newActiveVMs(),
		)

		Expect(handler.HandleMigrationSource(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalled).To(BeFalse())

		Expect(handler.HandleMigrationTarget(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalled).To(BeFalse())
	},
		Entry("When there are no networks",
			libvmi.New(libvmi.WithAutoAttachPodInterface(false)),
		),
		Entry("When an iface connected to pod network uses masquerade binding",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When an iface connected to pod network uses bridge binding",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("default")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When an iface connected to pod network uses tap attachment",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin("default", v1.PluginBinding{Name: "tap"})),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When an iface connected to pod network uses managedTap attachment",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin("default", v1.PluginBinding{Name: "managedTap"})),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When there is no iface connected to pod network",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary", "secondary-nad")),
			),
		),
	)

	DescribeTable("Should run passt repair on migration source", func(vmi *v1.VirtualMachineInstance) {
		passtRepairCalled := false
		fakeCommandWithCallCounter := func(string, *v1.VirtualMachineInstance, func(instance *v1.VirtualMachineInstance)) {
			passtRepairCalled = true
		}

		handler := passt.NewRepairManagerWithOptions(
			clusterConfigPasst,
			stubFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newActiveVMs(),
		)
		Expect(handler.HandleMigrationSource(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalled).To(BeTrue())
	},
		Entry("When an iface is connected to pod network using passt binding plugin",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When an iface is connected to Multus default network using passt binding plugin",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
				libvmi.WithNetwork(&v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "alternative",
							Default:     true,
						},
					},
				}),
			),
		),
	)

	DescribeTable("Should run passt repair on migration target", func(vmi *v1.VirtualMachineInstance) {
		passtRepairCalled := false
		fakeCommandWithCallCounter := func(s string, instance *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
			passtRepairCalled = true
		}

		handler := passt.NewRepairManagerWithOptions(
			clusterConfigPasst,
			stubFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newActiveVMs(),
		)

		Expect(handler.HandleMigrationTarget(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalled).To(BeTrue())
	},
		Entry("When an iface is connected to pod network using passt binding plugin",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When an iface is connected to Multus default network using passt binding plugin",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
				libvmi.WithNetwork(&v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "alternative",
							Default:     true,
						},
					},
				}),
			),
		),
	)

	vmi := libvmi.New(
		libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	)

	expectedError := errors.New("test function error")
	failingSocketDirFunc := func(*v1.VirtualMachineInstance) (string, error) {
		return "", expectedError
	}
	failingFindRepairSocketFunc := func(_ string) (string, error) {
		return "", expectedError
	}

	DescribeTable("HandleMigrationSource should return error", func(
		findRepairSocketFunc func(_ string) (string, error),
		dirFunc func(_ *v1.VirtualMachineInstance) (string, error),
	) {
		handler := passt.NewRepairManagerWithOptions(
			clusterConfigPasst,
			findRepairSocketFunc,
			stubCommand,
			newActiveVMs(),
		)
		Expect(handler.HandleMigrationSource(vmi, dirFunc)).To(MatchError(expectedError))
	},
		Entry(
			"When dirFunc fails",
			stubFindRepairSocketInDir,
			failingSocketDirFunc,
		),

		Entry(
			"When findRepairSocketFunc fails",
			failingFindRepairSocketFunc,
			stubSocketDir,
		),
	)

	It("HandleMigrationTarget should return error when dirFunc fails", func() {
		handler := passt.NewRepairManagerWithOptions(
			clusterConfigPasst,
			stubFindRepairSocketInDir,
			stubCommand,
			newActiveVMs(),
		)
		Expect(handler.HandleMigrationTarget(vmi, failingSocketDirFunc)).To(MatchError(expectedError))
	})

	It("Should not run HandleMigrationSource because it is already running", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		passtRepairCalledCounter := 0
		fakeCommandWithCallCounter := func(s string, vmi *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
			passtRepairCalledCounter++
		}

		handler := passt.NewRepairManagerWithOptions(
			clusterConfigPasst,
			stubFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newActiveVMs(),
		)
		Expect(passtRepairCalledCounter).To(Equal(0))
		Expect(handler.HandleMigrationSource(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalledCounter).To(Equal(1))

		Expect(handler.HandleMigrationSource(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalledCounter).To(Equal(1))
	})

	It("Should not run HandleMigrationTarget because it is already running", func() {
		passtRepairCalledCounter := 0
		fakeCommandWithCallCounter := func(_ string, _ *v1.VirtualMachineInstance, _ func(instance *v1.VirtualMachineInstance)) {
			passtRepairCalledCounter++
		}

		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		handler := passt.NewRepairManagerWithOptions(
			clusterConfigPasst,
			stubFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newActiveVMs(),
		)
		Expect(passtRepairCalledCounter).To(Equal(0))
		Expect(handler.HandleMigrationTarget(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalledCounter).To(Equal(1))

		Expect(handler.HandleMigrationTarget(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalledCounter).To(Equal(1))
	})
})

type stubClusterConfig struct {
	registeredPlugins map[string]v1.InterfaceBindingPlugin
}

func (s stubClusterConfig) GetNetworkBindings() map[string]v1.InterfaceBindingPlugin {
	return s.registeredPlugins
}

type activeVMs struct {
	running map[types.UID]struct{}
}

func (s *activeVMs) TestAndSetActive(vmi *v1.VirtualMachineInstance) bool {
	_, isActive := s.running[vmi.UID]
	if !isActive {
		s.running[vmi.UID] = struct{}{}
	}
	return isActive
}

func (s *activeVMs) SetInactive(_ *v1.VirtualMachineInstance) {}

func newActiveVMs() *activeVMs {
	return &activeVMs{running: map[types.UID]struct{}{}}
}

func stubFindRepairSocketInDir(string) (string, error) {
	return "", nil
}

func stubCommand(string, *v1.VirtualMachineInstance, func(instance *v1.VirtualMachineInstance)) {}

func stubSocketDir(*v1.VirtualMachineInstance) (string, error) {
	return "/var/run/passt", nil
}
