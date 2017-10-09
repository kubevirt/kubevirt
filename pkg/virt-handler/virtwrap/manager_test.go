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
	"github.com/jeevatkm/go-model"
	"github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
)

var _ = Describe("Manager", func() {
	var mockConn *cli.MockConnection
	var mockDomain *cli.MockVirDomain
	var ctrl *gomock.Controller
	var recorder *record.FakeRecorder
	var mockDetector *isolation.MockPodIsolationDetector
	testVmName := "testvm"
	testNamespace := "testnamespace"
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)

	virtShareDir := "/var/run/kubevirt"

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		recorder = record.NewFakeRecorder(10)
		mockDetector = isolation.NewMockPodIsolationDetector(ctrl)
		// Make sure that we always free the domain after use
		mockDomain.EXPECT().Free()
	})

	expectIsolationDetectionForVM := func(vm *v1.VirtualMachine) *api.DomainSpec {
		var domainSpec api.DomainSpec
		Expect(model.Copy(&domainSpec, vm.Spec.Domain)).To(BeEmpty())

		namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
		domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
		pidFile := virtShareDir + "/qemu-pids/" + namespace + "_" + domain

		domainSpec.Name = testDomainName
		domainSpec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
		domainSpec.QEMUCmd = &api.Commandline{
			QEMUEnv: []api.Env{
				{Name: "SLICE", Value: "dfd"},
				{Name: "CONTROLLERS", Value: "a,b"},
				{Name: "PIDNS", Value: "/proc/1234/ns/pid"},
				{Name: "PIDFILE", Value: pidFile},
			},
		}
		isolationResult := isolation.NewIsolationResult(1234, "dfd", []string{"a", "b"})
		mockDetector.EXPECT().Detect(vm).Return(isolationResult, nil)
		return &domainSpec
	}

	Context("on successful VM sync", func() {
		It("should define and start a new VM", func() {
			vm := newVM(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			domainSpec := expectIsolationDetectionForVM(vm)

			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockConn.EXPECT().ListSecrets().Return(make([]string, 0, 0), nil)
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector, virtShareDir)
			newspec, err := manager.SyncVM(vm)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
			Expect(<-recorder.Events).To(ContainSubstring(v1.Created.String()))
			Expect(<-recorder.Events).To(ContainSubstring(v1.Started.String()))
			Expect(recorder.Events).To(BeEmpty())
		})
		It("should leave a defined and started VM alone", func() {
			vm := newVM(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVM(vm)
			xml, err := xml.Marshal(domainSpec)

			mockConn.EXPECT().ListSecrets().Return(make([]string, 0, 0), nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector, virtShareDir)
			newspec, err := manager.SyncVM(vm)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
			Expect(recorder.Events).To(BeEmpty())
		})
		table.DescribeTable("should try to start a VM in state",
			func(state libvirt.DomainState) {
				vm := newVM(testNamespace, testVmName)
				domainSpec := expectIsolationDetectionForVM(vm)
				xml, err := xml.Marshal(domainSpec)

				mockConn.EXPECT().ListSecrets().Return(make([]string, 0, 0), nil)
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
				mockDomain.EXPECT().Create().Return(nil)
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector, virtShareDir)
				newspec, err := manager.SyncVM(vm)
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
			domainSpec := expectIsolationDetectionForVM(vm)
			xml, err := xml.Marshal(domainSpec)

			mockConn.EXPECT().ListSecrets().Return(make([]string, 0, 0), nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector, virtShareDir)
			newspec, err := manager.SyncVM(vm)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
			Expect(<-recorder.Events).To(ContainSubstring(v1.Resumed.String()))
			Expect(recorder.Events).To(BeEmpty())
		})
	})
	Context("on successful VM kill", func() {
		table.DescribeTable("should try to undefine a VM in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().ListSecrets().Return(make([]string, 0, 0), nil)
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector, virtShareDir)
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
				mockConn.EXPECT().ListSecrets().Return(make([]string, 0, 0), nil)
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Destroy().Return(nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector, virtShareDir)
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

func newVM(namespace string, name string) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       v1.VMSpec{Domain: v1.NewMinimalDomainSpec()},
	}
}
