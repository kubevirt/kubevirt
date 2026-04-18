/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
	Context("should not run passt repair", func() {
		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			vmi = libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("default")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
		})

		It("when binding is not core passt binding", func() {
			passtRepairCalled := false
			fakeCommandWithCallCounter := func(s string, instance *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
				passtRepairCalled = true
			}

			handler := passt.NewRepairManagerWithOptions(
				stubFindRepairSocketInDir,
				fakeCommandWithCallCounter,
				newActiveVMs(),
			)

			Expect(handler.HandleMigrationSource(vmi, stubSocketDir)).To(Succeed())
			Expect(passtRepairCalled).To(BeFalse())

			Expect(handler.HandleMigrationTarget(vmi, stubSocketDir)).To(Succeed())
			Expect(passtRepairCalled).To(BeFalse())
		})
	})

	DescribeTable("Should run passt repair on migration source", func(vmi *v1.VirtualMachineInstance) {
		passtRepairCalled := false
		fakeCommandWithCallCounter := func(string, *v1.VirtualMachineInstance, func(instance *v1.VirtualMachineInstance)) {
			passtRepairCalled = true
		}

		handler := passt.NewRepairManagerWithOptions(
			stubFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newActiveVMs(),
		)
		Expect(handler.HandleMigrationSource(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalled).To(BeTrue())
	},
		Entry("When an iface is connected to pod network using passt binding",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When an iface is connected to Multus default network using passt binding",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
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
			stubFindRepairSocketInDir,
			fakeCommandWithCallCounter,
			newActiveVMs(),
		)

		Expect(handler.HandleMigrationTarget(vmi, stubSocketDir)).To(Succeed())
		Expect(passtRepairCalled).To(BeTrue())
	},
		Entry("When an iface is connected to pod network using passt binding",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry("When an iface is connected to Multus default network using passt binding",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
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
		libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
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
			stubFindRepairSocketInDir,
			stubCommand,
			newActiveVMs(),
		)
		Expect(handler.HandleMigrationTarget(vmi, failingSocketDirFunc)).To(MatchError(expectedError))
	})

	It("Should not run HandleMigrationSource because it is already running", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		passtRepairCalledCounter := 0
		fakeCommandWithCallCounter := func(s string, vmi *v1.VirtualMachineInstance, f func(instance *v1.VirtualMachineInstance)) {
			passtRepairCalledCounter++
		}

		handler := passt.NewRepairManagerWithOptions(
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
			libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		handler := passt.NewRepairManagerWithOptions(
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
