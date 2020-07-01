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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package eventsclient

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"os"
	"reflect"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	v13 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("Notify", func() {

	Describe("Domain Events", func() {
		var err error
		var shareDir string
		var stop chan struct{}
		var stopped bool
		var deleteNotificationSent chan watch.Event
		var client *Notifier

		var mockDomain *cli.MockVirDomain
		var mockCon *cli.MockConnection
		var ctrl *gomock.Controller

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockCon = cli.NewMockConnection(ctrl)
			mockDomain = cli.NewMockVirDomain(ctrl)
			mockCon.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil).AnyTimes()

			stop = make(chan struct{})
			deleteNotificationSent = make(chan watch.Event, 100)
			stopped = false
			shareDir, err = ioutil.TempDir("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())
			client = NewNotifier()
		})

		AfterEach(func() {
			if stopped == false {
				close(stop)
			}
			os.RemoveAll(shareDir)
			ctrl.Finish()
		})

		Context("server", func() {
			table.DescribeTable("should accept Domain notify events", func(state libvirt.DomainState, event libvirt.DomainEventType, kubevirtState api.LifeCycle, kubeEventType watch.EventType) {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())

				mockDomain.EXPECT().GetState().Return(state, -1, nil)
				mockDomain.EXPECT().Free()
				mockDomain.EXPECT().GetName().Return("test", nil).AnyTimes()
				mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)
				mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).Return(`<kubevirt></kubevirt>`, nil)

				eventCallback(mockCon, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: event}}, client, deleteNotificationSent, nil, nil)

				select {
				case <-time.After(2 * time.Second):
					Expect(true).To(BeFalse(), "should not time out")
				case <-client.DomainEventStore.UpdateChan():
					event := client.DomainEventStore.Get().(*v13.DomainEventRequest)
					newDomain := &api.Domain{}
					Expect(json.Unmarshal(event.GetDomainJSON(), newDomain)).To(Succeed())
					newDomain.Spec.XMLName = xml.Name{}
					Expect(newDomain.Spec).To(Equal(domain.Spec))
					Expect(event.GetEventType()).To(Equal(string(kubeEventType)))
				}
			},
				table.Entry("modified for crashed VMIs", libvirt.DOMAIN_CRASHED, libvirt.DOMAIN_EVENT_CRASHED, api.Crashed, watch.Modified),
				table.Entry("modified for stopped VMIs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_SHUTDOWN, api.Shutoff, watch.Modified),
				table.Entry("modified for stopped VMIs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_STOPPED, api.Shutoff, watch.Modified),
				table.Entry("modified for running VMIs", libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_EVENT_STARTED, api.Running, watch.Modified),
				table.Entry("added for defined VMIs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_DEFINED, api.Shutoff, watch.Added),
			)
		})

		It("should receive a delete event when a VirtualMachineInstance is undefined",
			func() {
				mockDomain.EXPECT().Free()
				mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return("", libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_NOSTATE, -1, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
				mockDomain.EXPECT().GetName().Return("test", nil).AnyTimes()

				eventCallback(mockCon, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED}}, client, deleteNotificationSent, nil, nil)

				select {
				case <-time.After(2 * time.Second):
					Expect(true).To(BeFalse(), "should not time out")
				case <-client.DomainEventStore.UpdateChan():
					event := client.DomainEventStore.Get().(*v13.DomainEventRequest)
					newDomain := &api.Domain{}
					Expect(json.Unmarshal(event.GetDomainJSON(), newDomain)).To(Succeed())
					Expect(newDomain.Status.Status).To(Equal(api.NoState))
					Expect(newDomain.ObjectMeta.DeletionTimestamp).ToNot(BeNil())
					Expect(event.GetEventType()).To(Equal(string(watch.Modified)))
				}

				select {
				case <-time.After(2 * time.Second):
					Expect(true).To(BeFalse(), "should not time out")
				case <-deleteNotificationSent:
					// virt-launcher waits in a final delete notification to be sent before exiting.
				}
			})

		It("should update Interface status",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockDomain.EXPECT().Free()
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockDomain.EXPECT().GetName().Return("test", nil).AnyTimes()
				mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)
				mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).Return(`<kubevirt></kubevirt>`, nil)

				interfaceStatus := []api.InterfaceStatus{
					api.InterfaceStatus{
						Name: "test", Ip: "1.1.1.1/24", Mac: "1", InterfaceName: "eth1",
					},
					api.InterfaceStatus{
						Name: "test2",
					},
				}

				eventCallback(mockCon, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, interfaceStatus, nil)

				select {
				case <-time.After(2 * time.Second):
					Expect(true).To(BeFalse(), "should not time out")
				case <-client.DomainEventStore.UpdateChan():
					event := client.DomainEventStore.Get().(*v13.DomainEventRequest)
					newDomain := &api.Domain{}
					Expect(json.Unmarshal(event.GetDomainJSON(), newDomain)).To(Succeed())
					newInterfaceStatuses := newDomain.Status.Interfaces
					Expect(len(newInterfaceStatuses)).To(Equal(2))
					Expect(reflect.DeepEqual(interfaceStatus, newInterfaceStatuses)).To(BeTrue())
				}
			})

		It("should update Guest OS Info",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockDomain.EXPECT().Free()
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockDomain.EXPECT().GetName().Return("test", nil).AnyTimes()
				mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)
				mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).Return(`<kubevirt></kubevirt>`, nil)

				guestOsName := "TestGuestOS"
				osInfoStatus := api.GuestOSInfo{
					Name: guestOsName,
				}

				eventCallback(mockCon, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, nil, &osInfoStatus)

				select {
				case <-time.After(2 * time.Second):
					Expect(true).To(BeFalse(), "should not time out")
				case <-client.DomainEventStore.UpdateChan():
					event := client.DomainEventStore.Get().(*v13.DomainEventRequest)
					newDomain := &api.Domain{}
					Expect(json.Unmarshal(event.GetDomainJSON(), newDomain)).To(Succeed())
					newOSStatus := newDomain.Status.OSInfo
					Expect(reflect.DeepEqual(osInfoStatus, newOSStatus)).To(BeTrue())
				}
			})
	})

	Describe("K8s Events", func() {
		var err error
		var shareDir string
		var stop chan struct{}
		var stopped bool
		var client *Notifier
		var vmiStore cache.Store

		BeforeEach(func() {
			stop = make(chan struct{})
			stopped = false
			shareDir, err = ioutil.TempDir("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()

			client = NewNotifier()
		})

		AfterEach(func() {
			if stopped == false {
				close(stop)
			}
			os.RemoveAll(shareDir)
		})

		It("Should send a k8s event", func() {

			vmi := v1.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			err := client.EnqueueK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())
			select {
			case <-time.After(2 * time.Second):
				Expect(true).To(BeFalse(), "should not time out")
			case <-client.K8sEventStore.UpdateChan():
				event := client.K8sEventStore.Get().(*v13.K8SEventRequest)
				var k8sEvent k8sv1.Event
				Expect(json.Unmarshal(event.GetEventJSON(), &k8sEvent)).To(Succeed())
				Expect(k8sEvent.Type).To(Equal(eventType))
				Expect(k8sEvent.Reason).To(Equal(eventReason))
				Expect(k8sEvent.Message).To(Equal(eventMessage))
			}
		})
	})
})
