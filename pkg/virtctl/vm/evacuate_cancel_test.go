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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Evacuate cancel command", func() {
	var (
		ctrl         *gomock.Controller
		vmiInterface *kubecli.MockVirtualMachineInstanceInterface
		vmInterface  *kubecli.MockVirtualMachineInterface
		kubeClient   *k8sfake.Clientset
		virtClient   *kubecli.MockKubevirtClient
	)

	const (
		vmName  = "testvm"
		vmiName = "testvmi"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.MockKubevirtClientInstance = virtClient
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig

		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		kubeClient = k8sfake.NewClientset()

		virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
	})

	It("should fail with missing arguments", func() {
		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel")
		Expect(cmd()).To(MatchError("accepts 2 arg(s), received 0"))
	})

	It("should fail on unsupported kind", func() {
		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "pod", "my-pod")
		Expect(cmd()).To(MatchError(`unsupported resource type "pod"`))
	})

	It("should cancel evacuation for VM", func() {
		vmInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmName, &v1.EvacuateCancelOptions{}).
			Return(nil).
			Times(1)

		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "vm", vmName)
		Expect(cmd()).To(Succeed())
	})

	It("should return error on VM evacuate cancel failure", func() {
		expectedErr := fmt.Errorf("failure on VM")
		vmInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmName, &v1.EvacuateCancelOptions{}).
			Return(expectedErr).
			Times(1)

		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "vm", vmName)
		Expect(cmd()).To(MatchError(expectedErr))
	})

	It("should cancel evacuation for VMI", func() {
		vmiInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmiName, &v1.EvacuateCancelOptions{}).
			Return(nil).
			Times(1)

		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "vmi", vmiName)
		Expect(cmd()).To(Succeed())
	})

	It("should print dry-run message", func() {
		cmd := testing.NewRepeatableVirtctlCommandWithOut("evacuate-cancel", "vmi", vmiName, "--dry-run")
		vmiInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmiName, &v1.EvacuateCancelOptions{
				DryRun: []string{k8smetav1.DryRunAll},
			}).Return(nil)

		bytes, err := cmd()
		Expect(err).ToNot(HaveOccurred())
		Expect(string(bytes)).To(ContainSubstring(fmt.Sprintf("VMI %s/%s was canceled evacuation", k8smetav1.NamespaceDefault, vmiName)))
	})

})
