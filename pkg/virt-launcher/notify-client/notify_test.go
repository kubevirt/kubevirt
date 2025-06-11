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

package eventsclient

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	api2 "kubevirt.io/client-go/api"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	"kubevirt.io/kubevirt/pkg/testutils"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("Notify", func() {

	Describe("Domain Events", func() {

		var eventChan chan watch.Event
		var deleteNotificationSent chan watch.Event
		var client *Notifier

		var mockLibvirt *testing.Libvirt
		var e *eventCaller

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockLibvirt = testing.NewLibvirt(ctrl)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()

			stop := make(chan struct{})
			eventChan = make(chan watch.Event, 100)
			deleteNotificationSent = make(chan watch.Event, 100)
			stopped := false
			shareDir, err := os.MkdirTemp("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())
			e = &eventCaller{}

			go func() {
				notifyserver.RunServer(shareDir, stop, eventChan, nil, nil)
			}()

			client = NewNotifier(shareDir)

			DeferCleanup(
				func() {
					if stopped == false {
						close(stop)
					}
					client.Close()
					os.RemoveAll(shareDir)
				},
			)
		})

		metadataCache := func() *metadata.Cache { return metadata.NewCache() }

		Context("server", func() {
			DescribeTable("should accept Domain notify events", func(state libvirt.DomainState, event libvirt.DomainEventType, kubevirtState api.LifeCycle, kubeEventType watch.EventType) {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())

				mockLibvirt.DomainEXPECT().GetState().Return(state, -1, nil)
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: event}}, client, deleteNotificationSent, nil, nil, nil, nil, metadataCache())

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, ok := event.Object.(*api.Domain)
					newDomain.Spec.XMLName = xml.Name{}
					Expect(ok).To(BeTrue(), "should typecase domain")
					Expect(equality.Semantic.DeepEqual(domain.Spec, newDomain.Spec)).To(BeTrue())
					Expect(event.Type).To(Equal(kubeEventType))
				}
				Expect(timedOut).To(BeFalse(), "should not time out")
			},
				Entry("modified for crashed VMIs", libvirt.DOMAIN_CRASHED, libvirt.DOMAIN_EVENT_CRASHED, api.Crashed, watch.Modified),
				Entry("modified for stopped VMIs with shutoff reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_SHUTDOWN, api.Shutoff, watch.Modified),
				Entry("modified for stopped VMIs with stopped reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_STOPPED, api.Shutoff, watch.Modified),
				Entry("modified for running VMIs", libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_EVENT_STARTED, api.Running, watch.Modified),
				Entry("added for defined VMIs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_DEFINED, api.Shutoff, watch.Added),
			)
		})

		It("should receive a delete event when a VirtualMachineInstance is undefined",
			func() {
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return("", libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_NOSTATE, -1, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED}}, client, deleteNotificationSent, nil, nil, nil, nil, metadataCache())

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case e := <-eventChan:
					Expect(e.Object.(*api.Domain).Status.Status).To(Equal(api.NoState))
					Expect(e.Object.(*api.Domain).ObjectMeta.DeletionTimestamp).ToNot(BeNil())
					Expect(e.Type).To(Equal(watch.Modified))

				}
				Expect(timedOut).To(BeFalse())

				select {
				case <-timeout:
					timedOut = true
				case <-deleteNotificationSent:
					// virt-launcher waits in a final delete notification to be sent before exiting.
				}
				Expect(timedOut).To(BeFalse())
			})

		It("should update Interface status",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				interfaceStatus := []api.InterfaceStatus{
					{
						Ip: "1.1.1.1/24", Mac: "1", InterfaceName: "eth1",
					},
				}

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, interfaceStatus, nil, nil, nil, metadataCache())

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, _ := event.Object.(*api.Domain)
					newInterfaceStatuses := newDomain.Status.Interfaces
					Expect(newInterfaceStatuses).To(HaveLen(1))
					Expect(equality.Semantic.DeepEqual(interfaceStatus, newInterfaceStatuses)).To(BeTrue())
				}
				Expect(timedOut).To(BeFalse())
			})

		It("should update Guest OS Info",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				guestOsName := "TestGuestOS"
				osInfoStatus := api.GuestOSInfo{
					Name: guestOsName,
				}

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, nil, &osInfoStatus, nil, nil, metadataCache())

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, _ := event.Object.(*api.Domain)
					newOSStatus := newDomain.Status.OSInfo
					Expect(equality.Semantic.DeepEqual(osInfoStatus, newOSStatus)).To(BeTrue())
				}
				Expect(timedOut).To(BeFalse())
			})

		It("should update Guest FSFreeze status",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				fsFrozenStatus := "frozen"
				fsFreezeStatus := api.FSFreeze{
					Status: fsFrozenStatus,
				}

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, nil, nil, nil, &fsFreezeStatus, metadataCache())

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, _ := event.Object.(*api.Domain)
					newFSFreezeStatus := newDomain.Status.FSFreezeStatus
					Expect(equality.Semantic.DeepEqual(fsFreezeStatus, newFSFreezeStatus)).To(BeTrue())
				}
				Expect(timedOut).To(BeFalse())
			})
	})

	Describe("K8s Events", func() {
		var err error
		var shareDir string
		var stop chan struct{}
		var stopped bool
		var eventChan chan watch.Event
		var deleteNotificationSent chan watch.Event
		var client *Notifier
		var recorder *record.FakeRecorder
		var vmiStore cache.Store
		var e *eventCaller

		BeforeEach(func() {
			stop = make(chan struct{})
			eventChan = make(chan watch.Event, 100)
			deleteNotificationSent = make(chan watch.Event, 100)
			stopped = false
			shareDir, err = os.MkdirTemp("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			recorder = record.NewFakeRecorder(10)
			recorder.IncludeObject = true
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()
			e = &eventCaller{}

			go func() {
				notifyserver.RunServer(shareDir, stop, eventChan, recorder, vmiStore)
			}()

			time.Sleep(1 * time.Second)

			client = NewNotifier(shareDir)
		})

		AfterEach(func() {
			if stopped == false {
				close(stop)
			}
			client.Close()
			os.RemoveAll(shareDir)
		})

		It("Should send a k8s event", func() {

			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			err := client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

			event := <-recorder.Events
			Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
		})

		It("Should generate a k8s event on IO errors", func() {
			faultDisk := []libvirt.DomainDiskError{
				{
					Disk:  "vda",
					Error: libvirt.DOMAIN_DISK_ERROR_NO_SPACE,
				},
			}
			domain := api.NewMinimalDomain("test")
			domain.Status.Reason = api.ReasonPausedIOError
			x, err := xml.Marshal(domain.Spec)
			Expect(err).ToNot(HaveOccurred())

			ctrl := gomock.NewController(GinkgoT())
			mockLibvirt := testing.NewLibvirt(ctrl)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_IOERROR), nil)
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)
			mockLibvirt.DomainEXPECT().GetDiskErrors(uint32(0)).Return(faultDisk, nil)

			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)
			eventType := "Warning"
			eventReason := "IOerror"
			eventMessage := "VM Paused due to not enough space on volume: "
			metadataCache := metadata.NewCache()
			e.eventCallback(mockLibvirt.VirtConnection, domain, libvirtEvent{}, client, deleteNotificationSent, nil, nil, vmi, nil, metadataCache)
			event := <-recorder.Events
			Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
		})

	})

	Describe("Version mismatch", func() {

		var err error
		var ctrl *gomock.Controller
		var infoClient *info.MockNotifyInfoClient

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			infoClient = info.NewMockNotifyInfoClient(ctrl)
		})

		It("Should report error when server version mismatches", func() {

			fakeResponse := info.NotifyInfoResponse{
				SupportedNotifyVersions: []uint32{42},
			}
			infoClient.EXPECT().Info(gomock.Any(), gomock.Any()).Return(&fakeResponse, nil)

			By("Initializing the notifier")
			_, err = negotiateVersion(infoClient)

			Expect(err).To(HaveOccurred(), "Should have returned error about incompatible versions")
			Expect(err.Error()).To(ContainSubstring("no compatible version found"), "Expected error message to contain 'no compatible version found'")

		})
	})
})
