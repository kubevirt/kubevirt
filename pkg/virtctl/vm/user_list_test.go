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

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("User list command", func() {
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller
	const vmName = "testvm"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	It("should fail with missing input parameters", func() {
		cmd := clientcmd.NewRepeatableVirtctlCommand("userlist")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("argument validation failed"))
	})

	It("should fail with non existing VM", func() {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)

		vmiInterface.EXPECT().UserList(context.Background(), vmName).Return(v1.VirtualMachineInstanceGuestOSUserList{}, fmt.Errorf("an error on the server (\"virtualmachineinstance.kubevirt.io \"testvm\" not found\") has prevented the request from succeeding")).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand("userlist", vmName)
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("Error listing users of VirtualMachineInstance testvm, an error on the server (\"virtualmachineinstance.kubevirt.io \"testvm\" not found\") has prevented the request from succeeding"))
	})

	It("should fail when vm has no userlist data", func() {
		vm := kubecli.NewMinimalVM(vmName)

		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)

		vmiInterface.EXPECT().UserList(context.Background(), vm.Name).Return(v1.VirtualMachineInstanceGuestOSUserList{}, fmt.Errorf("an error on the server (\"Operation cannot be fulfilled on virtualmachineinstance.kubevirt.io \"testvm\": VMI does not have guest agent connected\") has prevented the request from succeeding")).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand("userlist", vm.Name)
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("Error listing users of VirtualMachineInstance testvm, an error on the server (\"Operation cannot be fulfilled on virtualmachineinstance.kubevirt.io \"testvm\": VMI does not have guest agent connected\") has prevented the request from succeeding"))
	})

	It("should return userlist data", func() {
		vm := kubecli.NewMinimalVM(vmName)
		userList := v1.VirtualMachineInstanceGuestOSUserList{
			Items: []v1.VirtualMachineInstanceGuestOSUser{
				{
					UserName: "TEST",
				},
			},
		}

		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)

		vmiInterface.EXPECT().UserList(context.Background(), vm.Name).Return(userList, nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand("userlist", vm.Name)
		Expect(cmd()).To(Succeed())
	})
})
