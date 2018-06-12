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

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network"
)

var _ = Describe("Manager", func() {
	var mockConn *cli.MockConnection
	var mockDomain *cli.MockVirDomain
	var ctrl *gomock.Controller
	testVmName := "testvmi"
	testNamespace := "testnamespace"
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		// Make sure that we always free the domain after use
		mockDomain.EXPECT().Free()
	})

	expectIsolationDetectionForVMI := func(vmi *v1.VirtualMachineInstance) *api.DomainSpec {
		domain := &api.Domain{}
		c := &api.ConverterContext{
			VirtualMachine: vmi,
			UseEmulation:   true,
		}
		Expect(api.Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c)).To(Succeed())
		api.SetObjectDefaults_Domain(domain)

		return &domain.Spec
	}

	Context("on successful VirtualMachineInstance sync", func() {
		It("should define and start a new VirtualMachineInstance", func() {
			StubOutNetworkForTest()
			vmi := newVMI(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			domainSpec := expectIsolationDetectionForVMI(vmi)

			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
		})
		It("should leave a defined and started VirtualMachineInstance alone", func() {
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.Marshal(domainSpec)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
		})
		table.DescribeTable("should try to start a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				vmi := newVMI(testNamespace, testVmName)
				domainSpec := expectIsolationDetectionForVMI(vmi)
				xml, err := xml.Marshal(domainSpec)

				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
				mockDomain.EXPECT().Create().Return(nil)
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
				manager, _ := NewLibvirtDomainManager(mockConn)
				newspec, err := manager.SyncVMI(vmi, true)
				Expect(newspec).ToNot(BeNil())
				Expect(err).To(BeNil())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		It("should resume a paused VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.Marshal(domainSpec)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
		})
	})
	Context("on successful VirtualMachineInstance kill", func() {
		table.DescribeTable("should try to undefine a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn)
				err := manager.KillVMI(newVMI(testNamespace, testVmName))
				Expect(err).To(BeNil())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		table.DescribeTable("should try to destroy and undefine a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Destroy().Return(nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn)
				err := manager.KillVMI(newVMI(testNamespace, testVmName))
				Expect(err).To(BeNil())
			},
			table.Entry("running", libvirt.DOMAIN_RUNNING),
			table.Entry("paused", libvirt.DOMAIN_PAUSED),
		)
	})

	table.DescribeTable("on successful list all domains",
		func(state libvirt.DomainState, kubevirtState api.LifeCycle) {

			mockDomain.EXPECT().GetState().Return(state, -1, nil)
			mockDomain.EXPECT().GetName().Return("test", nil)
			x, err := xml.Marshal(api.NewMinimalDomainSpec("test"))
			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).Return(string(x), nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(string(x), nil)
			mockConn.EXPECT().ListAllDomains(gomock.Eq(libvirt.CONNECT_LIST_DOMAINS_ACTIVE|libvirt.CONNECT_LIST_DOMAINS_INACTIVE)).Return([]cli.VirDomain{mockDomain}, nil)

			manager, _ := NewLibvirtDomainManager(mockConn)
			doms, err := manager.ListAllDomains()

			Expect(len(doms)).To(Equal(1))

			domain := doms[0]
			domain.Spec.XMLName = xml.Name{}

			Expect(&domain.Spec).To(Equal(api.NewMinimalDomainSpec("test")))
			Expect(domain.Status.Status).To(Equal(kubevirtState))
		},
		table.Entry("crashed", libvirt.DOMAIN_CRASHED, api.Crashed),
		table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF, api.Shutoff),
		table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN, api.Shutdown),
		table.Entry("unknown", libvirt.DOMAIN_NOSTATE, api.NoState),
		table.Entry("running", libvirt.DOMAIN_RUNNING, api.Running),
	)

	// TODO: test error reporting on non successful VirtualMachineInstance syncs and kill attempts

	AfterEach(func() {
		ctrl.Finish()
	})
})

func newVMI(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       v1.VirtualMachineInstanceSpec{Domain: v1.NewMinimalDomainSpec()},
	}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func StubOutNetworkForTest() {
	network.SetupPodNetwork = func(vm *v1.VirtualMachineInstance, domain *api.Domain) error { return nil }
}
