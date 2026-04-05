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

package eventsserver

import (
	"net"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("watchSocketFile", func() {
	var (
		tmpDir   string
		sockFile string
	)

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		sockFile = filepath.Join(tmpDir, "test.sock")

		socketCheckInterval = 100 * time.Millisecond
	})

	createSocket := func(path string) (net.Listener, uint64) {
		listener, err := net.Listen("unix", path)
		Expect(err).NotTo(HaveOccurred())
		ino, err := socketInode(path)
		Expect(err).NotTo(HaveOccurred())
		return listener, ino
	}

	It("should detect socket file deletion", func() {
		listener, ino := createSocket(sockFile)
		defer listener.Close()

		stopChan := make(chan struct{})
		socketGone := make(chan struct{})
		go watchSocketFile(sockFile, ino, stopChan, socketGone)

		Expect(os.Remove(sockFile)).To(Succeed())

		Eventually(socketGone).WithTimeout(time.Second).Should(BeClosed())
	})

	It("should detect socket file replacement", func() {
		listener, ino := createSocket(sockFile)
		listener.Close()

		newListener, newIno := createSocket(sockFile)
		defer newListener.Close()
		Expect(newIno).NotTo(Equal(ino))

		stopChan := make(chan struct{})
		socketGone := make(chan struct{})
		go watchSocketFile(sockFile, ino, stopChan, socketGone)

		Eventually(socketGone).WithTimeout(time.Second).Should(BeClosed())
	})

	It("should not signal when socket file is unchanged", func() {
		listener, ino := createSocket(sockFile)
		defer listener.Close()

		stopChan := make(chan struct{})
		socketGone := make(chan struct{})
		go watchSocketFile(sockFile, ino, stopChan, socketGone)

		Consistently(socketGone).WithTimeout(500 * time.Millisecond).ShouldNot(BeClosed())
		close(stopChan)
	})

	It("should exit cleanly when stopChan is closed", func() {
		listener, ino := createSocket(sockFile)
		defer listener.Close()

		stopChan := make(chan struct{})
		socketGone := make(chan struct{})
		go watchSocketFile(sockFile, ino, stopChan, socketGone)

		close(stopChan)
		Consistently(socketGone).WithTimeout(500 * time.Millisecond).ShouldNot(BeClosed())
	})
})

var _ = Describe("socketInode", func() {
	It("should return the inode of an existing file", func() {
		tmpDir := GinkgoT().TempDir()
		sockFile := filepath.Join(tmpDir, "test.sock")

		listener, err := net.Listen("unix", sockFile)
		Expect(err).NotTo(HaveOccurred())
		defer listener.Close()

		ino, err := socketInode(sockFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(ino).NotTo(BeZero())
	})

	It("should return error for non-existent file", func() {
		_, err := socketInode("/nonexistent/path/socket.sock")
		Expect(err).To(HaveOccurred())
	})
})
