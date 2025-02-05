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

package vm_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Object graph command", func() {
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInterface *kubecli.MockVirtualMachineInterface
	var ctrl *gomock.Controller
	const objectGraphCommand = "objectgraph"
	const vmName = "testvm"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
	})

	It("should fail with missing input parameters", func() {
		cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand)
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("accepts 1 arg(s), received 0"))
	})

	It("should fail with non existing vm", func() {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)

		vmInterface.EXPECT().ObjectGraph(context.Background(), vmName).Return(v1.ObjectGraphNodeList{}, fmt.Errorf("test-error")).Times(1)

		cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vmName)
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("Error listing object graph of VirtualMachine testvm, test-error\n"))
	})

	It("should return object graph", func() {
		vm := kubecli.NewMinimalVM(vmName)
		objectGraph := v1.ObjectGraphNodeList{
			Items: []v1.ObjectGraphNode{},
		}
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)

		vmInterface.EXPECT().ObjectGraph(context.Background(), vm.Name).Return(objectGraph, nil).Times(1)

		cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vm.Name)
		Expect(cmd()).To(Succeed())
	})

	It("Should return object graph from VMI", func() {
		vm := kubecli.NewMinimalVM(vmName)
		objectGraph := v1.ObjectGraphNodeList{
			Items: []v1.ObjectGraphNode{},
		}

		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)

		vmiInterface.EXPECT().ObjectGraph(context.Background(), vm.Name).Return(objectGraph, nil).Times(1)

		cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vm.Name, "--vmi")
		Expect(cmd()).To(Succeed())
	})
})
