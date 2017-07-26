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

package virtwrap

import (
	"encoding/xml"
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/designer"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

var _ = Describe("Manager", func() {
	var mockConn *MockConnection
	var mockDomain *MockVirDomain
	var ctrl *gomock.Controller
	var recorder *record.FakeRecorder
	testVmName := "testvm"
	testNamespace := "testnamespace"
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = NewMockConnection(ctrl)
		mockDomain = NewMockVirDomain(ctrl)
		recorder = record.NewFakeRecorder(10)
		// Make sure that we always free the domain after use
		mockDomain.EXPECT().Free()
	})

	Context("on successful VM sync", func() {
		It("should define and start a new VM", func() {
			vm := newVM(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			domainDesign := &designer.DomainDesign{
				Domain: &api.DomainSpec{
					Name: testDomainName,
				},
			}

			xml, err := xml.Marshal(domainDesign.Domain)
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder)
			newspec, err := manager.SyncVM(vm, domainDesign)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
			Expect(<-recorder.Events).To(ContainSubstring(v1.Created.String()))
			Expect(<-recorder.Events).To(ContainSubstring(v1.Started.String()))
			Expect(recorder.Events).To(BeEmpty())
		})
		It("should leave a defined and started VM alone", func() {
			vm := newVM(testNamespace, testVmName)
			domainDesign := &designer.DomainDesign{
				Domain: &api.DomainSpec{
					Name: testDomainName,
				},
			}

			xml, err := xml.Marshal(domainDesign.Domain)
			Expect(err).To(BeNil())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder)
			newspec, err := manager.SyncVM(vm, domainDesign)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
			Expect(recorder.Events).To(BeEmpty())
		})
		table.DescribeTable("should try to start a VM in state",
			func(state libvirt.DomainState) {
				vm := newVM(testNamespace, testVmName)
				domainDesign := &designer.DomainDesign{
					Domain: &api.DomainSpec{
						Name: testDomainName,
					},
				}

				xml, err := xml.Marshal(domainDesign.Domain)
				Expect(err).To(BeNil())

				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Create().Return(nil)
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder)
				newspec, err := manager.SyncVM(vm, domainDesign)
				Expect(newspec).ToNot(BeNil())
				Expect(err).To(BeNil())
				Expect(<-recorder.Events).To(ContainSubstring(v1.Started.String()))
				Expect(recorder.Events).To(BeEmpty())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		It("should resume a paused VM", func() {
			vm := newVM(testNamespace, testVmName)
			domainDesign := &designer.DomainDesign{
				Domain: &api.DomainSpec{
					Name: testDomainName,
				},
			}

			xml, err := xml.Marshal(domainDesign.Domain)
			Expect(err).To(BeNil())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder)
			newspec, err := manager.SyncVM(vm, domainDesign)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
			Expect(<-recorder.Events).To(ContainSubstring(v1.Resumed.String()))
			Expect(recorder.Events).To(BeEmpty())
		})
	})
	Context("on successful VM kill", func() {
		table.DescribeTable("should try to undefine a VM in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder)
				err := manager.KillVM(newVM(testNamespace, testVmName))
				Expect(err).To(BeNil())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		table.DescribeTable("should try to destroy and undefine a VM in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Destroy().Return(nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder)
				err := manager.KillVM(newVM(testNamespace, testVmName))
				Expect(err).To(BeNil())
				Expect(<-recorder.Events).To(ContainSubstring(v1.Stopped.String()))
				Expect(<-recorder.Events).To(ContainSubstring(v1.Deleted.String()))
				Expect(recorder.Events).To(BeEmpty())
			},
			table.Entry("running", libvirt.DOMAIN_RUNNING),
			table.Entry("paused", libvirt.DOMAIN_PAUSED),
		)
	})

	// TODO: test error reporting on non successful VM syncs and kill attempts

	AfterEach(func() {
		ctrl.Finish()
	})
})

func newVM(namespace string, name string) *v1.VM {
	return &v1.VM{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       v1.VMSpec{Domain: v1.NewMinimalDomainSpec()},
	}
}
