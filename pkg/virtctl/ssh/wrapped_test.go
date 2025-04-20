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

package ssh

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Wrapped SSH", func() {

	var fakeKind, fakeNamespace, fakeName string
	var ssh SSH

	BeforeEach(func() {
		fakeKind = "fake-kind"
		fakeNamespace = "fake-ns"
		fakeName = "fake-name"
		ssh = SSH{}
	})

	Context("buildSSHTarget", func() {

		It("with SSH username", func() {
			ssh.options = SSHOptions{SSHUsername: "testuser"}
			sshTarget := ssh.buildSSHTarget(fakeKind, fakeNamespace, fakeName)
			Expect(sshTarget[0]).To(Equal("testuser@fake-kind.fake-name.fake-ns"))
		})

		It("without SSH username", func() {
			sshTarget := ssh.buildSSHTarget(fakeKind, fakeNamespace, fakeName)
			Expect(sshTarget[0]).To(Equal("fake-kind.fake-name.fake-ns"))
		})

	})

	It("buildProxyCommandOption", func() {
		const sshPort = 12345
		proxyCommand := buildProxyCommandOption(fakeKind, fakeNamespace, fakeName, sshPort)
		expected := fmt.Sprintf("port-forward --stdio=true fake-kind/fake-name/fake-ns %d", sshPort)
		Expect(proxyCommand).To(ContainSubstring(expected))
	})

	It("RunLocalClient", func() {
		runCommand = func(cmd *exec.Cmd) error {
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Args).To(HaveLen(4))
			Expect(cmd.Args[0]).To(Equal("ssh"))
			Expect(cmd.Args[2]).To(Equal(buildProxyCommandOption(fakeKind, fakeNamespace, fakeName, ssh.options.SSHPort)))
			Expect(cmd.Args[3]).To(Equal(ssh.buildSSHTarget(fakeKind, fakeNamespace, fakeName)[0]))

			return nil
		}

		ssh.options = DefaultSSHOptions()
		ssh.options.SSHPort = 12345
		clientArgs := ssh.buildSSHTarget(fakeKind, fakeNamespace, fakeName)
		err := RunLocalClient(fakeKind, fakeNamespace, fakeName, &ssh.options, clientArgs)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
