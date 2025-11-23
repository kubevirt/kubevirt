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

package passtrefactor_test

import (
	"errors"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/passtrefactor"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

type repairHandlerStub struct {
	isRunCommandCalled bool
	isFindSocketCalled bool
	findSocketError    error
	sync.WaitGroup
}

func newRepairHandlerStub() *repairHandlerStub {
	return &repairHandlerStub{}
}

func (r *repairHandlerStub) RunCommand(s string, instance *v1.VirtualMachineInstance) {
	r.isRunCommandCalled = true
	r.Done()
}

func (r *repairHandlerStub) FindSocket(s string) (string, error) {
	r.isFindSocketCalled = true
	return "", r.findSocketError
}

type blockingRepairHandlerStub struct {
	repairStarted chan struct{} // Signals that repair execution has started
	blockCh       chan struct{} // Blocks repair execution until closed
	callCount     int
}

func newBlockingRepairHandlerStub() *blockingRepairHandlerStub {
	return &blockingRepairHandlerStub{
		repairStarted: make(chan struct{}, 10),
		blockCh:       make(chan struct{}),
	}
}

func (b *blockingRepairHandlerStub) RunCommand(s string, instance *v1.VirtualMachineInstance) {
	b.callCount++
	b.repairStarted <- struct{}{}
	<-b.blockCh
}

func (b *blockingRepairHandlerStub) FindSocket(s string) (string, error) {
	return "", nil
}

var _ = Describe("Passt Repair Handler", func() {
	vmi := libvmi.New(
		libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	)

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
		handler := newRepairHandlerStub()
		repairController := passtrefactor.NewPasstRepairControllerWithOptions(
			handler,
			clusterConfigMultiPlugin,
		)

		err := repairController.Run(vmi, false, stubFindRepairSocketInDir)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(handler.isFindSocketCalled).To(BeFalse())
		Expect(handler.isRunCommandCalled).To(BeFalse())
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

	DescribeTable("Should run passt repair on migration target", func(vmi *v1.VirtualMachineInstance) {

		handler := newRepairHandlerStub()
		repairController := passtrefactor.NewPasstRepairControllerWithOptions(
			handler,
			clusterConfigPasst,
		)
		handler.Add(1)
		err := repairController.Run(vmi, true, stubFindRepairSocketInDir)
		handler.Wait()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(handler.isFindSocketCalled).To(BeTrue())
		Expect(handler.isRunCommandCalled).To(BeTrue())

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

	It("controller should return error when socketDirFunc fails", func() {

		expectedError := errors.New("test function error")
		failingSocketDirFunc := func(*v1.VirtualMachineInstance) (string, error) {
			return "", expectedError
		}

		handler := newRepairHandlerStub()
		repairController := passtrefactor.NewPasstRepairControllerWithOptions(
			handler,
			clusterConfigPasst,
		)

		err := repairController.Run(vmi, false, failingSocketDirFunc)
		Expect(err).To(MatchError(expectedError))
	})

	It("controller should return error when FindSocket fails", func() {

		handler := newRepairHandlerStub()
		expectedError := errors.New("findSocket error")
		handler.findSocketError = expectedError
		repairController := passtrefactor.NewPasstRepairControllerWithOptions(
			handler,
			clusterConfigPasst,
		)

		err := repairController.Run(vmi, true, stubFindRepairSocketInDir)
		Expect(err).To(MatchError(expectedError))
	})

	It("Should not run command because it is already running", func() {
		handler := newBlockingRepairHandlerStub()
		repairController := passtrefactor.NewPasstRepairControllerWithOptions(
			handler,
			clusterConfigPasst,
		)

		err := repairController.Run(vmi, true, stubFindRepairSocketInDir)
		Expect(err).ShouldNot(HaveOccurred())

		<-handler.repairStarted

		Expect(handler.callCount).To(Equal(1))

		By("Attempting to run a second command while first is still running")
		err = repairController.Run(vmi, true, stubFindRepairSocketInDir)
		Expect(err).ShouldNot(HaveOccurred())

		By("Verifying second command was blocked - counter remains at 1")
		Consistently(func() int {
			return handler.callCount
		}).WithTimeout(200*time.Millisecond).WithPolling(10*time.Millisecond).
			Should(Equal(1), "second repair should not have started")

		close(handler.blockCh)

		Expect(handler.callCount).To(Equal(1))
	})
})

type stubClusterConfig struct {
	registeredPlugins map[string]v1.InterfaceBindingPlugin
}

func (s stubClusterConfig) GetNetworkBindings() map[string]v1.InterfaceBindingPlugin {
	return s.registeredPlugins
}

func stubFindRepairSocketInDir(*v1.VirtualMachineInstance) (string, error) {
	return "", nil
}
