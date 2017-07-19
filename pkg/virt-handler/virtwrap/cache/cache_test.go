/*
 * This file is part of the kubevirt project
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

package cache

import (
	"github.com/golang/mock/gomock"
	"github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

var _ = Describe("Cache", func() {
	var mockConn *virtwrap.MockConnection
	var mockDomain *virtwrap.MockVirDomain
	var ctrl *gomock.Controller

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = virtwrap.NewMockConnection(ctrl)
		mockDomain = virtwrap.NewMockVirDomain(ctrl)
	})

	Context("on syncing with libvirt", func() {
		table.DescribeTable("should receive a VM through the initial listing of domains",
			func(state libvirt.DomainState, kubevirtState api.LifeCycle) {
				mockConn.EXPECT().DomainEventLifecycleRegister(gomock.Any()).Return(nil)
				mockDomain.EXPECT().GetState().Return(state, -1, nil)
				mockDomain.EXPECT().GetName().Return("test", nil)
				mockDomain.EXPECT().GetUUIDString().Return("1235", nil)
				mockConn.EXPECT().ListAllDomains(gomock.Eq(libvirt.CONNECT_LIST_DOMAINS_ACTIVE|libvirt.CONNECT_LIST_DOMAINS_INACTIVE)).Return([]virtwrap.VirDomain{mockDomain}, nil)

				informer, err := NewSharedInformer(mockConn)
				Expect(err).To(BeNil())
				stopChan := make(chan struct{})

				go informer.Run(stopChan)

				cache.WaitForCacheSync(stopChan, informer.HasSynced)
				obj, exists, err := informer.GetStore().GetByKey("default/test")
				Expect(err).To(BeNil())
				Expect(exists).To(BeTrue())

				domain := obj.(*api.Domain)

				Expect(domain.Status.Status).To(Equal(kubevirtState))
				close(stopChan)
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED, api.Crashed),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF, api.Shutoff),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN, api.Shutdown),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE, api.NoState),
			table.Entry("running", libvirt.DOMAIN_RUNNING, api.Running),
		)
		table.DescribeTable("should receive non delete evens of type",
			func(state libvirt.DomainState, event libvirt.DomainEventType, kubevirtState api.LifeCycle, kubeEventType watch.EventType) {
				mockDomain.EXPECT().GetState().Return(state, -1, nil)
				mockDomain.EXPECT().GetName().Return("test", nil)
				mockDomain.EXPECT().GetUUIDString().Return("1235", nil)

				watcher := &DomainWatcher{make(chan watch.Event, 1)}
				callback(mockDomain, &libvirt.DomainEventLifecycle{Event: event}, watcher.C)

				e := <-watcher.C

				Expect(e.Object.(*api.Domain).Status.Status).To(Equal(kubevirtState))
				Expect(e.Type).To(Equal(kubeEventType))
			},
			table.Entry("modified for crashed VMs", libvirt.DOMAIN_CRASHED, libvirt.DOMAIN_EVENT_CRASHED, api.Crashed, watch.Modified),
			table.Entry("modified for stopped VMs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_SHUTDOWN, api.Shutoff, watch.Modified),
			table.Entry("modified for stopped VMs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_STOPPED, api.Shutoff, watch.Modified),
			table.Entry("modified for running VMs", libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_EVENT_STARTED, api.Running, watch.Modified),
			table.Entry("added for defined VMs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_DEFINED, api.Shutoff, watch.Added),
		)
		It("should receive a delete event when a VM is undefined",
			func() {
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_NOSTATE, -1, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
				mockDomain.EXPECT().GetName().Return("test", nil)
				mockDomain.EXPECT().GetUUIDString().Return("1235", nil)

				watcher := &DomainWatcher{make(chan watch.Event, 1)}
				callback(mockDomain, &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED}, watcher.C)

				e := <-watcher.C

				Expect(e.Object.(*api.Domain).Status.Status).To(Equal(api.NoState))
				Expect(e.Type).To(Equal(watch.Deleted))
			})
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
