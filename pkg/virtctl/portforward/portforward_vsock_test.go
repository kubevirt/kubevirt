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
	"io"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

type fakeVsockStreamer struct {
	streamErr error
	conn      net.Conn
}

func (f *fakeVsockStreamer) Stream(_ kvcorev1.StreamOptions) error { return f.streamErr }
func (f *fakeVsockStreamer) AsConn() net.Conn                      { return f.conn }

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
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
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

	It("rejects UDP ports", func() {
		Expect(vsockPortForwardCommand("vmi/testvmi", "udp/8080:9090")()).
			To(MatchError(ContainSubstring(`VSOCK does not support protocol "udp"`)))
	})

	DescribeTable("rejects a target port outside the valid uint16 range",
		func(remotePort string) {
			Expect(vsockPortForwardCommand("vmi/testvmi", "8080:"+remotePort)()).
				To(MatchError(ContainSubstring("port must be between 1 and")))
		},
		Entry("zero", "0"),
		Entry("beyond uint16 max", "65536"),
		Entry("negative", "-1"),
	)

	It("proxies data bidirectionally through VSOCK when binding a local port", func() {
		vsockSide, testSide := net.Pipe()
		defer testSide.Close()

		useTLS := true
		vmiInterface.EXPECT().VSOCK("testvmi", &v1.VSOCKOptions{
			TargetPort: 9090,
			UseTLS:     &useTLS,
		}).Return(&fakeVsockStreamer{conn: vsockSide}, nil)

		l, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		localPort := l.Addr().(*net.TCPAddr).Port
		l.Close()

		go func() {
			defer GinkgoRecover()
			testing.NewRepeatableVirtctlCommand(
				"port-forward", "vmi/testvmi",
				fmt.Sprintf("%d:9090", localPort),
				"--vsock=true",
				"--stdio=false",
				"--vsock-tls=true",
			)()
		}()

		var conn net.Conn
		Eventually(func() error {
			var dialErr error
			conn, dialErr = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
			return dialErr
		}).Should(Succeed())
		defer conn.Close()

		_, err = conn.Write([]byte("ping"))
		Expect(err).NotTo(HaveOccurred())

		buf := make([]byte, 4)
		_, err = io.ReadFull(testSide, buf)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buf)).To(Equal("ping"))

		_, err = testSide.Write([]byte("pong"))
		Expect(err).NotTo(HaveOccurred())

		buf = make([]byte, 4)
		_, err = io.ReadFull(conn, buf)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buf)).To(Equal("pong"))
	})
})
