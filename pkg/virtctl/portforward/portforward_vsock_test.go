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
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

type fakeVsockStreamer struct{}

func (f *fakeVsockStreamer) Stream(_ kvcorev1.StreamOptions) error { return nil }
func (f *fakeVsockStreamer) AsConn() net.Conn                      { return nil }

func vsockPortForwardCommand(target, portSpec string, extraArgs ...string) func() error {
	args := append([]string{"port-forward", target, portSpec, "--vsock=true", "--stdio=true"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommand(args...)
}

var _ = Describe("Port forward over VSOCK", func() {
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()
	})

	DescribeTable("forwards to the VSOCK subresource",
		func(target, portSpec, resolvedName string, useTLS bool, extraArgs ...string) {
			vmiInterface.EXPECT().VSOCK(resolvedName, &v1.VSOCKOptions{
				TargetPort: 9090,
				UseTLS:     &useTLS,
			}).Return(&fakeVsockStreamer{}, nil)

			Expect(vsockPortForwardCommand(target, portSpec, extraArgs...)()).To(Succeed())
		},
		Entry("vmi kind, TLS enabled", "vmi/testvmi", "8080:9090", "testvmi", true, "--vsock-tls=true"),
		Entry("vm kind, TLS enabled", "vm/testvm", "8080:9090", "testvm", true, "--vsock-tls=true"),
		Entry("vmi kind, TLS disabled", "vmi/testvmi", "8080:9090", "testvmi", false, "--vsock-tls=false"),
		Entry("vmi kind with explicit tcp protocol", "vmi/testvmi", "tcp/8080:9090", "testvmi", true, "--vsock-tls=true"),
	)

	It("rejects multiple ports with stdio", func() {
		Expect(vsockPortForwardCommand("vmi/testvmi", "8080:9090", "8081:9091")()).
			To(MatchError(ContainSubstring("only one port supported when forwarding to stdout")))
	})

	It("rejects UDP ports", func() {
		Expect(vsockPortForwardCommand("vmi/testvmi", "udp/8080:9090")()).
			To(MatchError(ContainSubstring(`forwarding protocol "udp" is not supported, only forwarding TCP is supported`)))
	})

})
