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

package plugins

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sidecar hooks", func() {

	Context("validateSocketPath", func() {
		It("should reject paths that don't exist on the filesystem", func() {
			err := validateSocketPath(pluginSocketBaseDir+"/my-plugin/hook.sock", "my-plugin")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stat socket path"))
		})

		It("should reject symlinks", func() {
			pluginDir := pluginSocketBaseDir + "/my-plugin"
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
			defer os.RemoveAll(pluginSocketBaseDir)

			target := filepath.Join(pluginDir, "real.sock")
			Expect(os.WriteFile(target, nil, 0600)).To(Succeed())
			link := filepath.Join(pluginDir, "link.sock")
			Expect(os.Symlink(target, link)).To(Succeed())

			err := validateSocketPath(link, "my-plugin")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("symlink"))
		})
	})

	Context("waitForSidecarSocket", func() {
		It("should return immediately when socket already exists", func() {
			socketDir := pluginSocketBaseDir + "/test-plugin"
			Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())
			defer os.RemoveAll(pluginSocketBaseDir)

			socketPath := filepath.Join(socketDir, "hook.sock")
			Expect(os.WriteFile(socketPath, nil, 0600)).To(Succeed())

			readinessTimeout := 2 * time.Second
			deadline := time.Now().Add(-1 * time.Second)
			Expect(waitForSidecarSocket(socketPath, deadline, readinessTimeout)).To(Succeed())
		})

		It("should return error when deadline passes and socket does not exist", func() {
			deadline := time.Now().Add(-1 * time.Second)
			readinessTimeout := 200 * time.Millisecond
			err := waitForSidecarSocket("/var/run/kubevirt-plugin/test-plugin/nonexistent.sock", deadline, readinessTimeout)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not ready"))
			Expect(err.Error()).To(ContainSubstring(readinessTimeout.String()))
		})

		It("should report errors while checking the socket path", func() {
			socketDir := filepath.Join(pluginSocketBaseDir, "test-plugin")
			Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())
			defer os.RemoveAll(pluginSocketBaseDir)

			notDirectory := filepath.Join(socketDir, "not-a-directory")
			Expect(os.WriteFile(notDirectory, nil, 0600)).To(Succeed())

			deadline := time.Now().Add(-1 * time.Second)
			err := waitForSidecarSocket(filepath.Join(notDirectory, "hook.sock"), deadline, 200*time.Millisecond)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("checking socket"))
		})

		It("should wait for socket to appear before deadline", func() {
			socketDir := pluginSocketBaseDir + "/test-plugin"
			Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())
			defer os.RemoveAll(pluginSocketBaseDir)

			socketPath := filepath.Join(socketDir, "hook.sock")
			go func() {
				time.Sleep(100 * time.Millisecond)
				Expect(os.WriteFile(socketPath, nil, 0600)).To(Succeed())
			}()

			readinessTimeout := 2 * time.Second
			deadline := time.Now().Add(readinessTimeout)
			Expect(waitForSidecarSocket(socketPath, deadline, readinessTimeout)).To(Succeed())
		})
	})

})
