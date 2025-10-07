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

//nolint:lll
package accesscredentials

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("AccessCredentials", func() {
	var mockLibvirt *testing.Libvirt
	var ctrl *gomock.Controller
	var manager *AccessCredentialManager
	var tmpDir string
	var lock sync.Mutex

	BeforeEach(func() {
		var err error
		ctrl = gomock.NewController(GinkgoT())
		mockLibvirt = testing.NewLibvirt(ctrl)

		manager = NewManager(mockLibvirt.VirtConnection, &lock, metadata.NewCache())
		manager.resyncCheckIntervalSeconds = 1
		tmpDir, err = os.MkdirTemp("", "credential-test")
		Expect(err).ToNot(HaveOccurred())
		unitTestSecretDir = tmpDir
	})
	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	expectIsolationDetectionForVMI := func(vmi *v1.VirtualMachineInstance) *api.DomainSpec {
		domain := &api.Domain{}
		c := &converter.ConverterContext{
			Architecture:   arch.NewConverter(runtime.GOARCH),
			VirtualMachine: vmi,
			AllowEmulation: true,
			SMBios:         &cmdv1.SMBios{},
		}
		Expect(converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
		api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

		return &domain.Spec
	}

	It("should handle qemu agent exec", func() {
		const domName = "some-domain"
		const command = "some-command"
		args := []string{"arg1", "arg2"}

		const expectedCmd = `{"execute": "guest-exec", "arguments": { "path": "some-command", "arg": [ "arg1", "arg2" ], "capture-output":true } }`
		const expectedStatusCmd = `{"execute": "guest-exec-status", "arguments": { "pid": 789 } }`

		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedCmd, domName).Return(`{"return":{"pid":789}}`, nil)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(`{"return":{"exitcode":0,"out-data":"c3NoIHNvbWVrZXkxMjMgdGVzdC1rZXkK","exited":true}}`, nil)

		Expect(manager.agentGuestExec(domName, command, args)).To(Equal("ssh somekey123 test-key\n"))
	})

	It("should handle dynamically updating user/password with qemu agent", func() {
		const domName = "some-domain"
		const user = "myuser"
		const password = "1234"

		mockLibvirt.ConnectionEXPECT().LookupDomainByName(domName).Return(mockLibvirt.VirtDomain, nil).Times(1)
		mockLibvirt.DomainEXPECT().Free()
		mockLibvirt.DomainEXPECT().SetUserPassword(user, password, libvirt.DomainSetUserPasswordFlags(0)).Times(1)

		Expect(manager.agentSetUserPassword(domName, user, password)).To(Succeed())
	})

	It("should handle dynamically updating ssh key with qemu agent", func() {
		const domName = "some-domain"
		const user = "someowner"

		authorizedKeys := []string{"ssh some injected key"}

		mockLibvirt.ConnectionEXPECT().LookupDomainByName(domName).Return(mockLibvirt.VirtDomain, nil).Times(1)
		mockLibvirt.DomainEXPECT().AuthorizedSSHKeysSet(user, authorizedKeys, gomock.Any()).Return(nil).Times(1)
		mockLibvirt.DomainEXPECT().Free().Times(1)

		Expect(manager.agentSetAuthorizedKeys(domName, user, authorizedKeys)).To(Succeed())
	})

	It("should dynamically update ssh key with old qemu agent", func() {
		const domName = "some-domain"
		const user = "someowner"
		const filePath = "/home/someowner/.ssh"

		authorizedKeys := []string{"ssh some injected key"}

		mockLibvirt.ConnectionEXPECT().LookupDomainByName(domName).Return(mockLibvirt.VirtDomain, nil).Times(1)
		// The AuthorizedSSHKeysSet method fails so a backward compatible code will be used.
		mockLibvirt.DomainEXPECT().AuthorizedSSHKeysSet(user, authorizedKeys, gomock.Any()).Return(libvirt.ERR_INTERNAL_ERROR).Times(1)
		mockLibvirt.DomainEXPECT().Free().Times(1)

		expectedOpenCmd := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s/authorized_keys", "mode":"r" } }`, filePath)
		expectedWriteOpenCmd := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s/authorized_keys", "mode":"w" } }`, filePath)
		const expectedOpenCmdRes = `{"return":1000}`

		existingKey := base64.StdEncoding.EncodeToString([]byte("ssh some existing key"))
		const expectedReadCmd = `{"execute": "guest-file-read", "arguments": { "handle": 1000 } }`
		expectedReadCmdRes := fmt.Sprintf(`{"return":{"count":24,"buf-b64": %q}}`, existingKey)

		mergedKeys := base64.StdEncoding.EncodeToString([]byte(strings.Join(authorizedKeys, "\n")))
		expectedWriteCmd := fmt.Sprintf(`{"execute": "guest-file-write", "arguments": { "handle": 1000, "buf-b64": %q } }`, mergedKeys)

		const expectedCloseCmd = `{"execute": "guest-file-close", "arguments": { "handle": 1000 } }`

		const expectedExecReturn = `{"return":{"pid":789}}`
		const expectedStatusCmd = `{"execute": "guest-exec-status", "arguments": { "pid": 789 } }`

		getentBase64Str := base64.StdEncoding.EncodeToString([]byte("someowner:x:1111:2222:Some Owner:/home/someowner:/bin/bash"))
		const expectedHomeDirCmd = `{"execute": "guest-exec", "arguments": { "path": "getent", "arg": [ "passwd", "someowner" ], "capture-output":true } }`
		expectedHomeDirCmdRes := fmt.Sprintf(`{"return":{"exitcode":0,"out-data":%q,"exited":true}}`, getentBase64Str)

		expectedMkdirCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "mkdir", "arg": [ "-p", %q ], "capture-output":true } }`, filePath)
		const expectedMkdirRes = `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedParentChownCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chown", "arg": [ "1111:2222", %q ], "capture-output":true } }`, filePath)
		const expectedParentChownRes = `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedParentChmodCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chmod", "arg": [ "700", %q ], "capture-output":true } }`, filePath)
		const expectedParentChmodRes = `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedFileChownCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chown", "arg": [ "1111:2222", "%s/authorized_keys" ], "capture-output":true } }`, filePath)
		const expectedFileChownRes = `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedFileChmodCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chmod", "arg": [ "600", "%s/authorized_keys" ], "capture-output":true } }`, filePath)
		const expectedFileChmodRes = `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		// Detect user home dir
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedHomeDirCmd, domName).Return(expectedExecReturn, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedHomeDirCmdRes, nil).Times(1)

		// Expected Read File
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedOpenCmd, domName).Return(expectedOpenCmdRes, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedReadCmd, domName).Return(expectedReadCmdRes, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedCloseCmd, domName).Return("", nil).Times(1)

		// Expected prepare directory
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedMkdirCmd, domName).Return(expectedExecReturn, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedMkdirRes, nil).Times(1)

		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedParentChownCmd, domName).Return(expectedExecReturn, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedParentChownRes, nil).Times(1)

		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedParentChmodCmd, domName).Return(expectedExecReturn, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedParentChmodRes, nil).Times(1)

		// Expected Write file
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedWriteOpenCmd, domName).Return(expectedOpenCmdRes, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedWriteCmd, domName).Return("", nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedCloseCmd, domName).Return("", nil).Times(1)

		// Expected set file permissions
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedFileChownCmd, domName).Return(expectedExecReturn, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedFileChownRes, nil).Times(1)

		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedFileChmodCmd, domName).Return(expectedExecReturn, nil).Times(1)
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedFileChmodRes, nil).Times(1)

		Expect(manager.agentSetAuthorizedKeys(domName, user, authorizedKeys)).To(Succeed())
	})

	It("should fail to update ssh key if both methods return error", func() {
		const domName = "some-domain"
		const user = "someowner"

		authorizedKeys := []string{"ssh some injected key"}

		mockLibvirt.ConnectionEXPECT().LookupDomainByName(domName).Return(mockLibvirt.VirtDomain, nil).Times(1)
		// The AuthorizedSSHKeysSet method fails so a backward compatible code will be used.
		mockLibvirt.DomainEXPECT().AuthorizedSSHKeysSet(user, authorizedKeys, gomock.Any()).Return(libvirt.ERR_INTERNAL_ERROR).Times(1)
		mockLibvirt.DomainEXPECT().Free().Times(1)

		// Detect user home dir
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return("", libvirt.ERR_INTERNAL_ERROR).AnyTimes()

		Expect(manager.agentSetAuthorizedKeys(domName, user, authorizedKeys)).
			To(MatchError(ContainSubstring("failed to set SSH keys")))
	})

	It("should support multiple ssh keys in one secret value", func() {
		const secretID = "some-secret-123"
		const user = "fakeuser"

		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.AccessCredentials = []v1.AccessCredential{{
			SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
				Source: v1.SSHPublicKeyAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: secretID,
					},
				},
				PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
						Users: []string{user},
					},
				},
			},
		}}

		secretDirs := getSecretDirs(vmi)
		Expect(secretDirs).To(HaveLen(1))

		secretDir := secretDirs[0]
		Expect(os.Mkdir(secretDir, 0o755)).To(Succeed())

		const authorizedKeys = "first key\nsecond key\n"
		Expect(os.WriteFile(filepath.Join(secretDirs[0], "authorized_keys"), []byte(authorizedKeys), 0o600)).To(Succeed())

		keysLoaded := make(chan struct{})

		domName := util.VMINamespaceKeyFunc(vmi)

		const cmdPing = `{"execute":"guest-ping"}`
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(cmdPing, domName).AnyTimes().Return("", nil)

		mockLibvirt.ConnectionEXPECT().LookupDomainByName(domName).Return(mockLibvirt.VirtDomain, nil).Times(1)
		mockLibvirt.DomainEXPECT().AuthorizedSSHKeysSet(user, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ string, keys []string, _ any) error {
			defer GinkgoRecover()

			Expect(keys).To(Equal([]string{"first key", "second key"}))

			close(keysLoaded)
			return nil
		})
		mockLibvirt.DomainEXPECT().Free().Times(1)

		Expect(manager.HandleQemuAgentAccessCredentials(vmi)).To(Succeed())
		defer manager.Stop()

		// Wait until ssh keys reload is detected
		Eventually(keysLoaded, 5*time.Second, 50*time.Millisecond).Should(BeClosed())
	})

	It("should trigger updating a credential when secret propagation change occurs.", func() {
		var err error

		const secretID = "some-secret"
		const user = "fakeuser"
		password := "fakepassword"

		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.AccessCredentials = []v1.AccessCredential{
			{
				UserPassword: &v1.UserPasswordAccessCredential{
					Source: v1.UserPasswordAccessCredentialSource{
						Secret: &v1.AccessCredentialSecretSource{
							SecretName: secretID,
						},
					},
					PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
						QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
					},
				},
			},
		}
		domName := util.VMINamespaceKeyFunc(vmi)

		manager.stopCh = make(chan struct{})
		manager.watcher, err = fsnotify.NewWatcher()
		Expect(err).ToNot(HaveOccurred())

		secretDirs := getSecretDirs(vmi)
		Expect(secretDirs).To(HaveLen(1))
		Expect(secretDirs[0]).To(Equal(fmt.Sprintf("%s/%s-access-cred", tmpDir, secretID)))

		for _, dir := range secretDirs {
			Expect(os.Mkdir(dir, 0o755)).To(Succeed())
			Expect(manager.watcher.Add(dir)).To(Succeed())
		}

		// Write the file
		Expect(os.WriteFile(filepath.Join(secretDirs[0], user), []byte(password), 0o600)).To(Succeed())

		const cmdPing = `{"execute":"guest-ping"}`
		mockLibvirt.ConnectionEXPECT().QemuAgentCommand(cmdPing, domName).AnyTimes().Return("", nil)

		domainSpec := expectIsolationDetectionForVMI(vmi)
		xml, err := xml.MarshalIndent(domainSpec, "", "\t")
		Expect(err).NotTo(HaveOccurred())

		mockLibvirt.DomainEXPECT().Free().AnyTimes()
		mockLibvirt.ConnectionEXPECT().LookupDomainByName(domName).AnyTimes().Return(mockLibvirt.VirtDomain, nil)
		mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
		mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Any()).AnyTimes().Return(string(xml), nil)

		mockLibvirt.ConnectionEXPECT().DomainDefineXML(gomock.Any()).AnyTimes().DoAndReturn(func(xml string) (cli.VirDomain, error) {
			const match = `			<accessCredential>
				<succeeded>true</succeeded>
			</accessCredential>`
			Expect(strings.Contains(xml, match)).To(BeTrue())
			return mockLibvirt.VirtDomain, nil
		})

		mockLibvirt.DomainEXPECT().SetUserPassword(user, password, libvirt.DomainSetUserPasswordFlags(0)).MinTimes(1)

		// and wait
		go func() {
			watchTimeout := time.NewTicker(2 * time.Second)
			defer watchTimeout.Stop()
			<-watchTimeout.C
			close(manager.stopCh)
		}()

		// TODO: Rewrite test to not call private functions.
		manager.watchSecrets(vmi)

		// And wait again after modifying file
		// Another execute command should occur with the updated password
		manager.stopCh = make(chan struct{})
		password += "morefake"
		Expect(os.WriteFile(filepath.Join(secretDirs[0], user), []byte(password), 0o600)).To(Succeed())

		mockLibvirt.DomainEXPECT().SetUserPassword(user, password, libvirt.DomainSetUserPasswordFlags(0)).MinTimes(1).Return(nil)

		go func() {
			watchTimeout := time.NewTicker(2 * time.Second)
			defer watchTimeout.Stop()
			<-watchTimeout.C
			close(manager.stopCh)
		}()

		manager.watchSecrets(vmi)
	})
})
