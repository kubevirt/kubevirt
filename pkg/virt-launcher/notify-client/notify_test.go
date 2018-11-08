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

	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("Domain notify", func() {
	var err error
	var shareDir string
	var stop chan struct{}
	var stopped bool
	var eventChan chan watch.Event
	var deleteNotificationSent chan watch.Event
	var client *DomainEventClient

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
			notifyserver.RunServer(shareDir, stop, eventChan)
		}()

		time.Sleep(1 * time.Second)
		client, err = NewDomainEventClient(shareDir)
		Expect(err).ToNot(HaveOccurred())
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

			libvirtEventCallback(mockCon, util.NewDomainFromName("test", "1234"), LibvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: event}}, client, deleteNotificationSent)

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
			Expect(timedOut).To(Equal(false))
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

			libvirtEventCallback(mockCon, util.NewDomainFromName("test", "1234"), LibvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED}}, client, deleteNotificationSent)

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
})
