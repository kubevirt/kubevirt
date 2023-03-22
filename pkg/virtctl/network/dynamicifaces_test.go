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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package network_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/network"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Dynamic Interface Attachment", func() {
	var (
		ctrl       *gomock.Controller
		kubeClient *fakek8sclient.Clientset
		vm         *kubecli.MockVirtualMachineInterface
		vmi        *kubecli.MockVirtualMachineInstanceInterface
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		DeferCleanup(ctrl.Finish)
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)

		kubeClient = fakek8sclient.NewSimpleClientset()
	})

	const (
		ifaceName                           = "pluggediface1"
		testNetworkAttachmentDefinitionName = "newnet"
		vmName                              = "myvm1"
	)

	DescribeTable("should fail when required input parameters are missing", func(cmdType string, args ...string) {
		cmd := clientcmd.NewVirtctlCommand(append([]string{cmdType}, args...)...)
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	},
		Entry("missing the VM name as parameter for the `AddInterface` cmd", network.HotplugCmdName),
		Entry("missing all required flags for the `AddInterface` cmd", network.HotplugCmdName, vmName),
		Entry("missing the network attachment definition name flag for the `AddInterface` cmd", network.HotplugCmdName, vmName, "--iface-name", ifaceName),
	)

	It("fails when the VM name argument is missing but all flags are provided", func() {
		cmd := clientcmd.NewVirtctlCommand(append([]string{network.HotplugCmdName}, requiredCmdFlags(testNetworkAttachmentDefinitionName, ifaceName)...)...)
		err := cmd.Execute()

		const missingArgError = "argument validation failed"
		Expect(err).To(MatchError(ContainSubstring(missingArgError)))
	})

	When("all the required input parameters are provided", func() {
		BeforeEach(func() {
			kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, errors.New("kubeClient command not mocked")
			})
		})

		It("hot-plug an interface works", func() {
			vmi = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			mockVMIAddInterfaceEndpoints(vmi, vmName, testNetworkAttachmentDefinitionName, ifaceName)

			cmdArgs := append(requiredCmdFlags(testNetworkAttachmentDefinitionName, ifaceName))
			cmd := clientcmd.NewVirtctlCommand(buildDynamicIfaceCmd(network.HotplugCmdName, vmName, cmdArgs...)...)
			Expect(cmd.Execute()).To(Succeed())
		})

		It("hot-plug an interface with the `--persist` option works", func() {
			vm = kubecli.NewMockVirtualMachineInterface(ctrl)
			mockVMAddInterfaceEndpoints(vm, vmName, testNetworkAttachmentDefinitionName, ifaceName)

			cmdArgs := append(requiredCmdFlags(testNetworkAttachmentDefinitionName, ifaceName), "--persist")
			cmd := clientcmd.NewVirtctlCommand(buildDynamicIfaceCmd(network.HotplugCmdName, vmName, cmdArgs...)...)
			Expect(cmd.Execute()).To(Succeed())
		})
	})
})

func buildDynamicIfaceCmd(cmdType string, vmName string, requiredCmdArgs ...string) []string {
	if cmdType == network.HotplugCmdName {
		return buildHotplugIfaceCmd(vmName, requiredCmdArgs...)
	}
	panic(fmt.Errorf("the only dynamic command currently implemented is for `addinterface`"))
}

func buildHotplugIfaceCmd(vmName string, requiredCmdArgs ...string) []string {
	return append([]string{network.HotplugCmdName, vmName}, requiredCmdArgs...)
}

func mockVMIAddInterfaceEndpoints(vmi *kubecli.MockVirtualMachineInstanceInterface, vmName string, networkAttachmentDefinitionName string, ifaceName string) {
	kubecli.MockKubevirtClientInstance.
		EXPECT().
		VirtualMachineInstance(k8smetav1.NamespaceDefault).
		Return(vmi).
		Times(1)
	vmi.EXPECT().AddInterface(context.Background(), vmName, gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) interface{} {
		Expect(arg2.(*v1.AddInterfaceOptions).NetworkAttachmentDefinitionName).To(Equal(networkAttachmentDefinitionName))
		Expect(arg2.(*v1.AddInterfaceOptions).Name).To(Equal(ifaceName))
		return nil
	})
}

func mockVMAddInterfaceEndpoints(vm *kubecli.MockVirtualMachineInterface, vmName string, networkAttachmentDefinitionName string, ifaceName string) {
	kubecli.MockKubevirtClientInstance.
		EXPECT().
		VirtualMachine(k8smetav1.NamespaceDefault).
		Return(vm).
		Times(1)
	vm.EXPECT().AddInterface(context.Background(), vmName, gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) interface{} {
		Expect(arg2.(*v1.AddInterfaceOptions).NetworkAttachmentDefinitionName).To(Equal(networkAttachmentDefinitionName))
		Expect(arg2.(*v1.AddInterfaceOptions).Name).To(Equal(ifaceName))
		return nil
	})
}

func requiredCmdFlags(networkAttachmentDefinitionName string, ifaceName string) []string {
	return []string{"--network-attachment-definition-name", networkAttachmentDefinitionName, "--iface-name", ifaceName}
}
