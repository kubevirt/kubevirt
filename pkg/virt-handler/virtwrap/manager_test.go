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
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
	cli "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/libvirt"
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

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		recorder = record.NewFakeRecorder(10)
		mockDetector = isolation.NewMockPodIsolationDetector(ctrl)
	})

	expectIsolationDetectionForVM := func(vm *v1.VirtualMachine) (*api.DomainSpec, *isolation.IsolationResult) {
		var domainSpec api.DomainSpec
		Expect(model.Copy(&domainSpec, vm.Spec.Domain)).To(BeEmpty())

		domainSpec.Name = testDomainName
		domainSpec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
		domainSpec.QEMUCmd = &api.Commandline{
			QEMUEnv: []api.Env{
				{Name: "SLICE", Value: "dfd"},
				{Name: "CONTROLLERS", Value: "a,b"},
			},
		}
		isolationResult := isolation.NewIsolationResult(1234, "dfd", []string{"a", "b"})
		return &domainSpec, isolationResult
	}

	Context("on successful VM sync", func() {
		It("hypervisor.UpdateGuest() should call libvirt.UpdateGuestSpec()", func() {
			vm := newVM(testNamespace, testVmName)
			domainSpec, isol := expectIsolationDetectionForVM(vm)
			xmls, err := xml.Marshal(domainSpec)

			var newSpec api.DomainSpec
			err = xml.Unmarshal([]byte(xmls), &newSpec)

			mockConn.EXPECT().UpdateGuestSpec(vm, isol).Return(&newSpec, nil)
			newspec, err := UpdateGuest(mockConn, vm, isol)
			Expect(newspec).ToNot(BeNil())
			Expect(err).To(BeNil())
		})
	})
	Context("on successful VM kill", func() {
		table.DescribeTable("should try to undefine a VM in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().ListSecrets().Return(make([]string, 0, 0), nil)
				mockConn.EXPECT().LookupGuestByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				mockDomain.EXPECT().Free()
				manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector)
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
				mockConn.EXPECT().LookupGuestByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Destroy().Return(nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				mockDomain.EXPECT().Free()
				manager, _ := NewLibvirtDomainManager(mockConn, recorder, mockDetector)
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
