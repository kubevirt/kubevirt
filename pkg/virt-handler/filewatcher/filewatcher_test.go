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

package filewatcher_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-handler/filewatcher"
)

var _ = Describe("Filewatcher", func() {
	var (
		path         string
		testfilePath string
		watcher      *filewatcher.FileWatcher
	)

	createFile := func() {
		_, err := os.Create(testfilePath)
		Expect(err).ToNot(HaveOccurred())
	}

	removeFile := func() {
		Expect(os.Remove(testfilePath)).To(Succeed())
	}

	BeforeEach(func() {
		path = GinkgoT().TempDir()
		testfilePath = filepath.Join(path, "testfile")
		watcher = filewatcher.New(testfilePath, 100*time.Millisecond)
	})

	AfterEach(func() {
		watcher.Close()
		Eventually(watcher.Events).Should(BeClosed())
		Eventually(watcher.Errors).Should(BeClosed())
	})

	It("Should detect a file being created", func() {
		watcher.Run()
		createFile()
		Eventually(watcher.Events).Should(Receive(Equal(filewatcher.Create)))
		removeFile()
	})

	It("Should detect a file being removed", func() {
		createFile()
		watcher.Run()
		removeFile()
		Eventually(watcher.Events).Should(Receive(Equal(filewatcher.Remove)))
	})

	It("Should detect the ino of a file changing", func() {
		createFile()
		watcher.Run()
		removeFile()
		createFile()
		Eventually(watcher.Events).Should(Receive(Equal(filewatcher.InoChange)))
	})

	It("Should detect nothing if file exists", func() {
		createFile()
		watcher.Run()
		Consistently(watcher.Events).ShouldNot(Receive())
	})

	It("Should not detect if other files are created or removed", func() {
		watcher.Run()
		otherfile := filepath.Join(path, "otherfile")
		_, err := os.Create(otherfile)
		Expect(err).ToNot(HaveOccurred())
		Consistently(watcher.Events).ShouldNot(Receive())
		Expect(os.Remove(otherfile)).To(Succeed())
		Consistently(watcher.Events).ShouldNot(Receive())
	})
})
