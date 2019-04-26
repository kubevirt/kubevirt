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
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	"kubevirt.io/kubevirt/pkg/testutils"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
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
		var eventChan chan watch.Event
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
			eventChan = make(chan watch.Event, 100)
			deleteNotificationSent = make(chan watch.Event, 100)
			stopped = false
			shareDir, err = ioutil.TempDir("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			go func() {
				notifyserver.RunServer(shareDir, stop, eventChan, nil, nil)
			}()

			time.Sleep(1 * time.Second)

			client, err = NewNotifier(shareDir)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if stopped == false {
				close(stop)
			}
			client.Close()
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

				eventCallback(mockCon, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: event}}, client, deleteNotificationSent, nil)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, ok := event.Object.(*api.Domain)
					newDomain.Spec.XMLName = xml.Name{}
					Expect(ok).To(Equal(true), "should typecase domain")
					Expect(reflect.DeepEqual(domain.Spec, newDomain.Spec)).To(Equal(true))
					Expect(event.Type).To(Equal(kubeEventType))
				}
				Expect(timedOut).To(Equal(false), "should not time out")
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

				eventCallback(mockCon, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED}}, client, deleteNotificationSent, nil)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case e := <-eventChan:
					Expect(e.Object.(*api.Domain).Status.Status).To(Equal(api.NoState))
					Expect(e.Type).To(Equal(watch.Deleted))
				}
				Expect(timedOut).To(Equal(false))

				select {
				case <-timeout:
					timedOut = true
				case <-deleteNotificationSent:
					// virt-launcher waits in a final delete notification to be sent before exiting.
				}
				Expect(timedOut).To(Equal(false))
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

				eventCallback(mockCon, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, &interfaceStatus)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, _ := event.Object.(*api.Domain)
					newInterfaceStatuses := newDomain.Status.Interfaces
					Expect(len(newInterfaceStatuses)).To(Equal(2))
					Expect(reflect.DeepEqual(interfaceStatus, newInterfaceStatuses)).To(Equal(true))
				}
				Expect(timedOut).To(Equal(false))
			})
	})

	Describe("K8s Events", func() {
		var err error
		var shareDir string
		var stop chan struct{}
		var stopped bool
		var eventChan chan watch.Event
		var client *Notifier
		var recorder *record.FakeRecorder
		var vmiStore cache.Store

		BeforeEach(func() {
			stop = make(chan struct{})
			eventChan = make(chan watch.Event, 100)
			stopped = false
			shareDir, err = ioutil.TempDir("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			recorder = record.NewFakeRecorder(10)
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()

			go func() {
				notifyserver.RunServer(shareDir, stop, eventChan, recorder, vmiStore)
			}()

			time.Sleep(1 * time.Second)

			client, err = NewNotifier(shareDir)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if stopped == false {
				close(stop)
			}
			client.Close()
			os.RemoveAll(shareDir)
		})

		It("Should send a k8s event", func(done Done) {

			vmi := v1.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			err := client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

			event := <-recorder.Events
			Expect(event).To(Equal(fmt.Sprintf("%s %s %s", eventType, eventReason, eventMessage)))
			close(done)
		}, 5)
	})

	Describe("Version mismatch", func() {

		var err error
		var ctrl *gomock.Controller
		var infoClient *info.MockNotifyInfoClient

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			infoClient = info.NewMockNotifyInfoClient(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("Should report error when server version mismatches", func() {

			fakeResponse := info.NotifyInfoResponse{
				SupportedNotifyVersions: []uint32{42},
			}
			infoClient.EXPECT().Info(gomock.Any(), gomock.Any()).Return(&fakeResponse, nil)

			By("Initializing the notifier")
			_, err = NewNotifierWithInfoClient(infoClient, nil)

			Expect(err).To(HaveOccurred(), "Should have returned error about incompatible versions")
			Expect(err.Error()).To(ContainSubstring("no compatible version found"), "Expected error message to contain 'no compatible version found'")

		})

	})
})
