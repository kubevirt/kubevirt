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

package rest

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"

	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var _ = Describe("Console handler", func() {
	var (
		ctrl                  *gomock.Controller
		mockIsolationDetector *isolation.MockPodIsolationDetector
		mockIsolationResult   *isolation.MockIsolationResult
		handler               *ConsoleHandler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockIsolationDetector = isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationResult = isolation.NewMockIsolationResult(ctrl)
		handler = NewConsoleHandler(mockIsolationDetector, nil, nil)
	})

	Describe("getUnixSocketPath", func() {
		It("should resolve the socket path via safepath", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")
			vmi.UID = "test-uid-1234"

			socketDir := filepath.Join(tmpDir, "run", "kubevirt-private", string(vmi.UID))
			Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())

			socketFile := filepath.Join(socketDir, "virt-serial0")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			defer l.Close()

			root, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(root, nil)

			p, err := handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).ToNot(HaveOccurred())
			Expect(unsafepath.UnsafeAbsolute(p.Raw())).To(Equal(filepath.Join(tmpDir, "run", "kubevirt-private", string(vmi.UID), "virt-serial0")))
		})

		It("should fail when Detect returns an error", func() {
			vmi := api.NewMinimalVMI("testvmi")
			mockIsolationDetector.EXPECT().Detect(vmi).Return(nil, fmt.Errorf("detect failed"))

			_, err := handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(MatchError("detect failed"))
		})

		It("should fail when MountRoot returns an error", func() {
			vmi := api.NewMinimalVMI("testvmi")
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(nil, fmt.Errorf("mountroot failed"))

			_, err := handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(MatchError("mountroot failed"))
		})

		It("should fail when the socket does not exist", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")
			vmi.UID = "test-uid-1234"

			root, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(root, nil)

			_, err = handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(HaveOccurred())
		})

		It("should reject a path containing a symlink", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")
			vmi.UID = "test-uid-1234"

			realDir := filepath.Join(tmpDir, "real-run", "kubevirt-private", string(vmi.UID))
			Expect(os.MkdirAll(realDir, 0755)).To(Succeed())

			socketFile := filepath.Join(realDir, "virt-serial0")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			defer l.Close()

			// Create "run" as a symlink to "real-run" to simulate an attack
			Expect(os.Symlink(filepath.Join(tmpDir, "real-run"), filepath.Join(tmpDir, "run"))).To(Succeed())

			root, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(root, nil)

			_, err = handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("unixSocketDialer", func() {
		It("should connect to a unix socket via safepath", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")

			socketFile := filepath.Join(tmpDir, "test.sock")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			defer l.Close()

			socketPath, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir, "test.sock")
			Expect(err).ToNot(HaveOccurred())

			dial := unixSocketDialer(vmi, socketPath)
			conn, err := dial()
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			conn.Close()
		})

		It("should fail when the socket path is invalid", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")

			// Create the file so safepath resolves, then close the listener
			// (Go's UnixListener unlinks the socket on Close)
			socketFile := filepath.Join(tmpDir, "gone.sock")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())

			socketPath, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir, "gone.sock")
			Expect(err).ToNot(HaveOccurred())

			l.Close()

			dial := unixSocketDialer(vmi, socketPath)
			_, err = dial()
			Expect(err).To(HaveOccurred())
		})
	})
})
