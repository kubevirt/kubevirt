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
	stubSocketDirFunc := func(_ *v1.VirtualMachineInstance) (string, error) {
		return "/var/run/passt", nil
	}

	DescribeTable("Should not run passt repair", func(vmi *v1.VirtualMachineInstance) {
		clusterConfig := stubClusterConfig{
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

		commandCallCounter := 0
		fakeCommandWithCallCounter := func(s string, instance *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
			commandCallCounter++
		}

		handler := passt.NewRepairManagerWithOptions(
			clusterConfig,
			fakeCreateShortenedSymlink,
			fakeFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newStubStore(),
		)

		Expect(handler.HandleMigrationSource(vmi, stubSocketDirFunc)).To(Succeed())
		Expect(commandCallCounter).To(Equal(0))

		Expect(handler.HandleMigrationTarget(vmi, stubSocketDirFunc)).To(Succeed())
		Expect(commandCallCounter).To(Equal(0))
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

	DescribeTable("Should run passt repair", func(vmi *v1.VirtualMachineInstance) {
		clusterConfig := stubClusterConfig{
			registeredPlugins: map[string]v1.InterfaceBindingPlugin{
				"passt": {
					SidecarImage:                "passt:latest",
					NetworkAttachmentDefinition: "default/passt-network",
				},
			},
		}

		commandCallCounter := 0
		fakeCommandWithCallCounter := func(s string, instance *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
			commandCallCounter++
		}

		handler := passt.NewRepairManagerWithOptions(
			clusterConfig,
			fakeCreateShortenedSymlink,
			fakeFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newStubStore(),
		)
		Expect(handler.HandleMigrationSource(vmi, stubSocketDirFunc)).To(Succeed())
		Expect(commandCallCounter).To(Equal(1))
		Expect(handler.HandleMigrationTarget(vmi, stubSocketDirFunc)).To(Succeed())
		Expect(commandCallCounter).To(Equal(2))
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

	Context("error handling", func() {
		clusterConfig := stubClusterConfig{
			registeredPlugins: map[string]v1.InterfaceBindingPlugin{
				"passt": {
					SidecarImage:                "passt:latest",
					NetworkAttachmentDefinition: "default/passt-network",
				},
			},
		}
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		expectedError := errors.New("test function error")
		failingSocketDirFunc := func(*v1.VirtualMachineInstance) (string, error) {
			return "", expectedError
		}
		failingCreateShortenedSymlinkFunc := func(_ string) (string, error) {
			return "", expectedError
		}
		failingFindRepairSocketFunc := func(_ string) (string, error) {
			return "", expectedError
		}

		DescribeTable("HandleMigrationSource should return error", func(
			createSymlinkFunc func(_ string) (string, error),
			findRepairSocketFunc func(_ string) (string, error),
			dirFunc func(_ *v1.VirtualMachineInstance) (string, error),
		) {
			handler := passt.NewRepairManagerWithOptions(
				clusterConfig,
				createSymlinkFunc,
				findRepairSocketFunc,
				fakeCommand,
				newStubStore(),
			)

			Expect(handler.HandleMigrationSource(vmi, dirFunc)).To(MatchError(expectedError))
		},
			Entry(
				"When createSymlinkFunc fails",
				failingCreateShortenedSymlinkFunc,
				fakeFindRepairSocketInDir,
				stubSocketDirFunc,
			),
			Entry(
				"When findRepairSocketFunc fails",
				fakeCreateShortenedSymlink,
				failingFindRepairSocketFunc,
				stubSocketDirFunc,
			),
			Entry(
				"When dirFunc fails",
				fakeCreateShortenedSymlink,
				fakeFindRepairSocketInDir,
				failingSocketDirFunc,
			),
		)

		DescribeTable("HandleMigrationTarget should return error", func(
			createSymlinkFunc func(_ string) (string, error),
			findRepairSocketFunc func(_ string) (string, error),
			dirFunc func(_ *v1.VirtualMachineInstance) (string, error),
		) {
			handler := passt.NewRepairManagerWithOptions(
				clusterConfig,
				createSymlinkFunc,
				findRepairSocketFunc,
				fakeCommand,
				newStubStore(),
			)

			Expect(handler.HandleMigrationTarget(vmi, dirFunc)).To(MatchError(expectedError))
		},
			Entry(
				"When createSymlinkFunc fails",
				failingCreateShortenedSymlinkFunc,
				fakeFindRepairSocketInDir,
				stubSocketDirFunc,
			),
			Entry(
				"When dirFunc fails",
				fakeCreateShortenedSymlink,
				fakeFindRepairSocketInDir,
				failingSocketDirFunc,
			),
		)
	})

	It("Should not run command because it is already running", func() {
		clusterConfig := stubClusterConfig{
			registeredPlugins: map[string]v1.InterfaceBindingPlugin{
				"passt": {
					SidecarImage:                "passt:latest",
					NetworkAttachmentDefinition: "default/passt-network",
				},
			},
		}
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		commandCallCounter := 0
		fakeCommandWithCallCounter := func(s string, instance *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
			commandCallCounter++
		}

		handler := passt.NewRepairManagerWithOptions(
			clusterConfig,
			fakeCreateShortenedSymlink,
			fakeFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newStubStoreWithActiveVMI(vmi.UID),
		)

		Expect(handler.HandleMigrationSource(vmi, stubSocketDirFunc)).To(Succeed())
		Expect(commandCallCounter).To(Equal(0))

		Expect(handler.HandleMigrationTarget(vmi, stubSocketDirFunc)).To(Succeed())
		Expect(commandCallCounter).To(Equal(0))
	})
})

type stubClusterConfig struct {
	registeredPlugins map[string]v1.InterfaceBindingPlugin
}

func (s stubClusterConfig) GetNetworkBindings() map[string]v1.InterfaceBindingPlugin {
	return s.registeredPlugins
}

type stubStore struct {
	running map[types.UID]struct{}
}

func (s *stubStore) TestAndSetActive(vmi *v1.VirtualMachineInstance) bool {
	_, isActive := s.running[vmi.UID]
	return isActive
}

func (s *stubStore) SetInactive(vmi *v1.VirtualMachineInstance) {
	delete(s.running, vmi.UID)
}

func newStubStore() *stubStore {
	return &stubStore{running: map[types.UID]struct{}{}}
}

func newStubStoreWithActiveVMI(uid types.UID) *stubStore {
	return &stubStore{running: map[types.UID]struct{}{uid: {}}}
}

func fakeCreateShortenedSymlink(string) (string, error) {
	return "", nil
}

func fakeFindRepairSocketInDir(string) (string, error) {
	return "", nil
}

func fakeCommand(string, *v1.VirtualMachineInstance, func(instance *v1.VirtualMachineInstance)) {}
