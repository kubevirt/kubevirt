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
package pipe

import (
	"io"
	"net"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var _ = Describe("InjectNotify", func() {

	var pod *isolation.MockIsolationResult
	var socketPath string

	BeforeEach(func() {
		tmp := GinkgoT().TempDir()
		safeTmp, err := safepath.JoinAndResolveWithRelativeRoot(tmp)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(tmp, "dir"), 0777)).To(Succeed())

		pod = isolation.NewMockIsolationResult(gomock.NewController(GinkgoT()))
		pod.EXPECT().MountRoot().Return(safeTmp, nil)

		socketPath = filepath.Join(tmp, "dir", "domain-notify-pipe.sock")
	})
	It("should return working listener", func() {
		listener, err := InjectNotify(pod, "dir", false)
		Expect(err).ToNot(HaveOccurred())
		defer listener.Close()

		resultChan := make(chan error)
		go func() {
			conn, err := listener.Accept()
			Expect(conn.Close()).To(Succeed())
			resultChan <- err
		}()

		conn, err := net.Dial("unix", socketPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(conn.Close()).To(Succeed())
		Eventually(resultChan).Should(Receive(BeNil()))
	})

	It("should make it accessible for nonroot", func() {
		original := diskutils.DefaultOwnershipManager
		DeferCleanup(func() {
			diskutils.DefaultOwnershipManager = original
		})
		ownershipManager := diskutils.NewMockOwnershipManagerInterface(gomock.NewController(GinkgoT()))
		diskutils.DefaultOwnershipManager = ownershipManager
		ownershipManager.EXPECT().SetFileOwnership(gomock.Any()).Times(1)

		listener, err := InjectNotify(pod, "dir", true)
		Expect(err).ToNot(HaveOccurred())
		defer listener.Close()
	})
})

var _ = Describe("Proxy", func() {
	It("should proxy", func() {
		pipeS, pipe := net.Pipe()
		notifyS, notify := net.Pipe()

		exit := make(chan struct{})
		go func() {
			Proxy(log.Log, pipeS, func() (net.Conn, error) { return notifyS, nil })
			exit <- struct{}{}
		}()

		_, err := pipe.Write([]byte("Hello"))
		Expect(err).ToNot(HaveOccurred())

		Expect(func() (string, error) {
			buf := make([]byte, 5)
			_, err := notify.Read(buf)
			return string(buf), err
		}()).Should(Equal("Hello"))

		_, err = notify.Write([]byte("Hello back"))
		Expect(err).ToNot(HaveOccurred())

		Expect(func() (string, error) {
			buf := make([]byte, 10)
			_, err := pipe.Read(buf)
			return string(buf), err
		}()).Should(Equal("Hello back"))

		// Closing the pipe should close the notify connection and proxy should exit
		Expect(pipe.Close()).To(Succeed())
		Eventually(exit).Should(Receive())
		Eventually(func() error {
			_, err := notify.Read(make([]byte, 5))
			return err
		}).Should(MatchError(io.EOF))
	})
})
