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

package domainpublisher_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	domainpublisher "kubevirt.io/kubevirt/pkg/virt-launcher/domain-publisher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

var _ = Describe("RegisterLibvirtEvents", func() {
	var (
		mockLibvirt *testing.Libvirt
		handler     *fakeHandler
	)

	BeforeEach(func() {
		mockLibvirt = testing.NewLibvirt(gomock.NewController(GinkgoT()))
		handler = &fakeHandler{delivered: make(chan any, 10)}
	})

	It("should deliver each libvirt event to the handler", func() {
		var (
			lifecycle              libvirt.DomainEventLifecycleCallback
			jobCompleted           libvirt.DomainEventJobCompletedCallback
			agentLifecycle         libvirt.DomainEventAgentLifecycleCallback
			ioError                libvirt.DomainEventIOErrorReasonCallback
			deviceAdded            libvirt.DomainEventDeviceAddedCallback
			deviceRemoved          libvirt.DomainEventDeviceRemovedCallback
			memoryDeviceSizeChange libvirt.DomainEventMemoryDeviceSizeChangeCallback
			reconnect              chan bool
		)
		conn := mockLibvirt.ConnectionEXPECT()
		conn.SetReconnectChan(gomock.Any()).Do(func(ch chan bool) { reconnect = ch })
		conn.DomainEventLifecycleRegister(gomock.Any()).Do(func(cb libvirt.DomainEventLifecycleCallback) { lifecycle = cb })
		conn.DomainEventJobCompletedRegister(gomock.Any()).Do(func(cb libvirt.DomainEventJobCompletedCallback) { jobCompleted = cb })
		conn.AgentEventLifecycleRegister(gomock.Any()).Do(func(cb libvirt.DomainEventAgentLifecycleCallback) { agentLifecycle = cb })
		conn.DomainEventIOErrorReasonRegister(gomock.Any()).Do(func(cb libvirt.DomainEventIOErrorReasonCallback) { ioError = cb })
		conn.DomainEventDeviceAddedRegister(gomock.Any()).Do(func(cb libvirt.DomainEventDeviceAddedCallback) { deviceAdded = cb })
		conn.DomainEventDeviceRemovedRegister(gomock.Any()).Do(func(cb libvirt.DomainEventDeviceRemovedCallback) { deviceRemoved = cb })
		conn.DomainEventMemoryDeviceSizeChangeRegister(gomock.Any()).Do(func(cb libvirt.DomainEventMemoryDeviceSizeChangeCallback) {
			memoryDeviceSizeChange = cb
		})

		Expect(domainpublisher.RegisterLibvirtEvents(mockLibvirt.VirtConnection, handler)).To(Succeed())

		lifecycleEvent := &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_STARTED}
		lifecycle(nil, nil, lifecycleEvent)
		Expect(handler.delivered).To(Receive(BeIdenticalTo(lifecycleEvent)))

		jobCompletedEvent := &libvirt.DomainEventJobCompleted{}
		jobCompleted(nil, nil, jobCompletedEvent)
		Expect(handler.delivered).To(Receive(BeIdenticalTo(jobCompletedEvent)))

		agentEvent := &libvirt.DomainEventAgentLifecycle{State: libvirt.CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_CONNECTED}
		agentLifecycle(nil, nil, agentEvent)
		Expect(handler.delivered).To(Receive(BeIdenticalTo(agentEvent)))

		ioErrorEvent := &libvirt.DomainEventIOErrorReason{Action: libvirt.DOMAIN_EVENT_IO_ERROR_PAUSE}
		ioError(nil, nil, ioErrorEvent)
		Expect(handler.delivered).To(Receive(BeIdenticalTo(ioErrorEvent)))

		deviceAdded(nil, nil, &libvirt.DomainEventDeviceAdded{})
		Expect(handler.delivered).To(Receive(Equal(resync)))

		deviceRemoved(nil, nil, &libvirt.DomainEventDeviceRemoved{})
		Expect(handler.delivered).To(Receive(Equal(resync)))

		memoryDeviceSizeChange(nil, nil, &libvirt.DomainEventMemoryDeviceSizeChange{})
		Expect(handler.delivered).To(Receive(Equal(resync)))

		reconnect <- true
		Eventually(handler.delivered).Should(Receive(Equal(resync)))
	})

	It("should return a failed registration", func() {
		registerErr := errors.New("registration failed")
		conn := mockLibvirt.ConnectionEXPECT()
		conn.SetReconnectChan(gomock.Any())
		conn.DomainEventLifecycleRegister(gomock.Any()).Return(registerErr)
		conn.DomainEventJobCompletedRegister(gomock.Any())
		conn.AgentEventLifecycleRegister(gomock.Any())
		conn.DomainEventIOErrorReasonRegister(gomock.Any())
		conn.DomainEventDeviceAddedRegister(gomock.Any())
		conn.DomainEventDeviceRemovedRegister(gomock.Any())
		conn.DomainEventMemoryDeviceSizeChangeRegister(gomock.Any())

		Expect(domainpublisher.RegisterLibvirtEvents(mockLibvirt.VirtConnection, handler)).To(MatchError(registerErr))
	})
})

const resync = "resync"

type fakeHandler struct {
	delivered chan any
}

func (f *fakeHandler) OnLifecycle(event *libvirt.DomainEventLifecycle)       { f.delivered <- event }
func (f *fakeHandler) OnJobCompleted(event *libvirt.DomainEventJobCompleted) { f.delivered <- event }
func (f *fakeHandler) OnAgentLifecycle(event *libvirt.DomainEventAgentLifecycle) {
	f.delivered <- event
}
func (f *fakeHandler) OnIOError(event *libvirt.DomainEventIOErrorReason) { f.delivered <- event }
func (f *fakeHandler) Resync()                                           { f.delivered <- resync }
