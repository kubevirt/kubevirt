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
 */

package reset_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/reset"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Resetting", func() {
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := testing.NewRepeatableVirtctlCommand(reset.COMMAND_RESET)
			Expect(cmd()).To(MatchError(ContainSubstring("received 0")))
		})
	})

	It("should reset VMI", func() {
		vmiName := "resetable-vmi"

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Reset(context.Background(), vmiName).Return(nil).Times(1)

		cmd := testing.NewRepeatableVirtctlCommand(reset.COMMAND_RESET, vmiName)
		Expect(cmd()).To(Succeed())
	})
	It("should fail reset of VMI when server returns VMI not found", func() {
		vmiName := "resetable-vmi"

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Reset(context.Background(), vmiName).Return(fmt.Errorf("vmi not found")).Times(1)

		cmd := testing.NewRepeatableVirtctlCommand(reset.COMMAND_RESET, vmiName)
		Expect(cmd()).To(MatchError(ContainSubstring("not found")))
	})
})
