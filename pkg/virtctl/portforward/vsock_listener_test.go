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

package portforward

import (
	"errors"
	"fmt"
	"io"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

// pipeStreamer adapts a net.Conn (one end of a net.Pipe) to the StreamInterface
// returned by the VSOCK subresource, standing in for the real VSOCK connection.
type pipeStreamer struct {
	conn net.Conn
}

func (p *pipeStreamer) Stream(_ kvcorev1.StreamOptions) error { return nil }
func (p *pipeStreamer) AsConn() net.Conn                      { return p.conn }

func freeLocalPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("unexpected listener address type")
	}
	return addr.Port, nil
}

var _ = Describe("VSOCK local port forwarding", func() {
	It("binds a local TCP port and proxies accepted connections through VSOCK", func() {
		ctrl := gomock.NewController(GinkgoT())
		vmiInterface := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vsockSide, testSide := net.Pipe()
		defer testSide.Close()

		useTLS := true
		vmiInterface.EXPECT().VSOCK("testvmi", &v1.VSOCKOptions{
			TargetPort: 9090,
			UseTLS:     &useTLS,
		}).Return(&pipeStreamer{conn: vsockSide}, nil)

		localPort, err := freeLocalPort()
		Expect(err).NotTo(HaveOccurred())

		address, err := net.ResolveIPAddr("", "127.0.0.1")
		Expect(err).NotTo(HaveOccurred())

		forwarder := portForwarder{
			kind:      "vmi",
			namespace: "default",
			name:      "testvmi",
			resource:  vsockResource{iface: vmiInterface, useTLS: true},
		}
		port := forwardedPort{local: localPort, remote: 9090, protocol: protocolTCP}
		Expect(forwarder.startForwarding(address, port)).To(Succeed())

		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
		Expect(err).NotTo(HaveOccurred())
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
