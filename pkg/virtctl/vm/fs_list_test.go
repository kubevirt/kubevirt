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

var _ = Describe("FS list command", func() {
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
		cmd := clientcmd.NewRepeatableVirtctlCommand("fslist")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("argument validation failed"))
	})

	It("should fail with non existing vm", func() {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)

		vmiInterface.EXPECT().FilesystemList(context.Background(), vmName).Return(v1.VirtualMachineInstanceFileSystemList{}, fmt.Errorf("Error listing filesystems of VirtualMachineInstance testvm, an error on the server (\"virtualmachineinstance.kubevirt.io \"testvm\" not found\") has prevented the request from succeeding")).Times(1)

		cmd := clientcmd.NewVirtctlCommand("fslist", vmName)
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("Error listing filesystems of VirtualMachineInstance testvm, Error listing filesystems of VirtualMachineInstance testvm, an error on the server (\"virtualmachineinstance.kubevirt.io \"testvm\" not found\") has prevented the request from succeeding"))
	})

	It("should return filesystem  data", func() {
		vm := kubecli.NewMinimalVM(vmName)
		fsList := v1.VirtualMachineInstanceFileSystemList{
			Items: []v1.VirtualMachineInstanceFileSystem{
				{
					DiskName: "TEST",
				},
			},
		}

		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)

		vmiInterface.EXPECT().FilesystemList(context.Background(), vm.Name).Return(fsList, nil).Times(1)

		cmd := clientcmd.NewVirtctlCommand("fslist", vm.Name)
		Expect(cmd.Execute()).To(Succeed())
	})
})
