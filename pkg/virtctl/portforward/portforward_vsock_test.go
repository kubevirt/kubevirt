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

package portforward_test

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

type fakeVsockStreamer struct {
	streamErr error
}

func (f *fakeVsockStreamer) Stream(_ kvcorev1.StreamOptions) error { return f.streamErr }
func (f *fakeVsockStreamer) AsConn() net.Conn                      { return nil }

var _ = Describe("Port forward over VSOCK", func() {
	var ctrl *gomock.Controller
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	DescribeTable("forwards to the VSOCK subresource once the VMI is confirmed running",
		func(target, resolvedName string, useTLS bool, extraArgs ...string) {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), resolvedName, metav1.GetOptions{}).Return(&v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
			}, nil)
			vmiInterface.EXPECT().VSOCK(resolvedName, &v1.VSOCKOptions{
				TargetPort: 9090,
				UseTLS:     &useTLS,
			}).Return(&fakeVsockStreamer{}, nil)

			args := append([]string{"port-forward", target, "8080:9090", "--vsock=true", "--stdio=true"}, extraArgs...)
			Expect(testing.NewRepeatableVirtctlCommand(args...)()).To(Succeed())
		},
		Entry("vmi kind, default TLS", "vmi/testvmi", "testvmi", true),
		Entry("vm kind, default TLS", "vm/testvm", "testvm", true),
		Entry("vmi kind, TLS disabled via --vsock-tls=false", "vmi/testvmi", "testvmi", false, "--vsock-tls=false"),
	)

	DescribeTable("returns an error when the VMI is not usable",
		func(vmi *v1.VirtualMachineInstance, getErr error, expectedErrSubstring string) {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), "testvmi", metav1.GetOptions{}).Return(vmi, getErr)

			Expect(testing.NewRepeatableVirtctlCommand(
				"port-forward", "vmi/testvmi", "8080:9090", "--vsock=true", "--stdio=true",
			)()).To(MatchError(ContainSubstring(expectedErrSubstring)))
		},
		Entry("VMI not found", nil, fmt.Errorf("not found"), "failed to find VirtualMachineInstance"),
		Entry("VMI not running", &v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{Phase: v1.Scheduling},
		}, nil, "is not running (phase: Scheduling)"),
	)

	It("rejects UDP ports", func() {
		Expect(testing.NewRepeatableVirtctlCommand(
			"port-forward", "vmi/testvmi", "udp/8080:9090", "--vsock=true", "--stdio=true",
		)()).To(MatchError(ContainSubstring(`VSOCK does not support protocol "udp"`)))
	})

	DescribeTable("rejects a target port outside the valid uint16 range",
		func(remotePort string) {
			Expect(testing.NewRepeatableVirtctlCommand(
				"port-forward", "vmi/testvmi", "8080:"+remotePort, "--vsock=true", "--stdio=true",
			)()).To(MatchError(ContainSubstring("port must be between 1 and")))
		},
		Entry("zero", "0"),
		Entry("beyond uint16 max", "65536"),
	)
})
