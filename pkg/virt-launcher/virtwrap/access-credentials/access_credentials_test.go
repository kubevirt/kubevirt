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
 * Copyright 2020 Red Hat, Inc.
 *
 */

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

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/mock/gomock"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("AccessCredentials", func() {
	var mockConn *cli.MockConnection
	var mockDomain *cli.MockVirDomain
	var ctrl *gomock.Controller
	var manager *AccessCredentialManager
	var tmpDir string
	var lock sync.Mutex

	BeforeEach(func() {
		var err error
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)

		manager = NewManager(mockConn, &lock, metadata.NewCache())
		manager.resyncCheckIntervalSeconds = 1
		tmpDir, err = os.MkdirTemp("", "credential-test")
		Expect(err).ToNot(HaveOccurred())
		unitTestSecretDir = tmpDir
	})
	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	expectIsolationDetectionForVMI := func(vmi *v1.VirtualMachineInstance) *api.DomainSpec {
		domain := &api.Domain{}
		c := &converter.ConverterContext{
			Architecture:   runtime.GOARCH,
			VirtualMachine: vmi,
			AllowEmulation: true,
			SMBios:         &cmdv1.SMBios{},
		}
		Expect(converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
		api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

		return &domain.Spec
	}

	It("should handle qemu agent exec", func() {
		domName := "some-domain"
		command := "some-command"
		args := []string{"arg1", "arg2"}

		expectedCmd := `{"execute": "guest-exec", "arguments": { "path": "some-command", "arg": [ "arg1", "arg2" ], "capture-output":true } }`
		expectedStatusCmd := `{"execute": "guest-exec-status", "arguments": { "pid": 789 } }`

		mockConn.EXPECT().QemuAgentCommand(expectedCmd, domName).Return(`{"return":{"pid":789}}`, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(`{"return":{"exitcode":0,"out-data":"c3NoIHNvbWVrZXkxMjMgdGVzdC1rZXkK","exited":true}}`, nil)

		Expect(manager.agentGuestExec(domName, command, args)).To(Equal("ssh somekey123 test-key\n"))
	})

	It("should handle dynamically updating user/password with qemu agent", func() {

		domName := "some-domain"
		password := "1234"
		user := "myuser"
		base64Str := base64.StdEncoding.EncodeToString([]byte(password))
		cmdSetPassword := fmt.Sprintf(`{"execute":"guest-set-user-password", "arguments": {"username":"%s", "password": "%s", "crypted": false }}`, user, base64Str)
		mockConn.EXPECT().QemuAgentCommand(cmdSetPassword, domName).Return("", nil)

		Expect(manager.agentSetUserPassword(domName, user, password)).To(Succeed())
	})

	It("should handle dynamically updating ssh key with qemu agent", func() {
		domName := "some-domain"
		user := "someowner"

		authorizedKeys := []string{"ssh some injected key"}

		mockConn.EXPECT().LookupDomainByName(domName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().AuthorizedSSHKeysSet(user, authorizedKeys, gomock.Any()).Return(nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)

		Expect(manager.agentSetAuthorizedKeys(domName, user, authorizedKeys)).To(Succeed())
	})

	It("should support multiple ssh keys in one secret value", func() {
		secretID := "some-secret-123"
		user := "fakeuser"

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
		Expect(os.Mkdir(secretDir, 0755)).To(Succeed())

		authorizedKeys := "first key\nsecond key\n"
		Expect(os.WriteFile(filepath.Join(secretDirs[0], "authorized_keys"), []byte(authorizedKeys), 0644)).To(Succeed())

		keysLoaded := make(chan struct{})

		domName := util.VMINamespaceKeyFunc(vmi)

		cmdPing := `{"execute":"guest-ping"}`
		mockConn.EXPECT().QemuAgentCommand(cmdPing, domName).AnyTimes().Return("", nil)

		mockConn.EXPECT().LookupDomainByName(domName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().AuthorizedSSHKeysSet(user, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ string, keys []string, _ any) error {
			defer GinkgoRecover()

			Expect(keys).To(Equal([]string{"first key", "second key"}))

			close(keysLoaded)
			return nil
		})
		mockDomain.EXPECT().Free().Times(1)

		Expect(manager.HandleQemuAgentAccessCredentials(vmi)).To(Succeed())
		DeferCleanup(func() {
			manager.Stop()
		})

		// Wait until ssh keys reload is detected
		Eventually(keysLoaded, 5*time.Second, 50*time.Millisecond).Should(BeClosed())
	})

	It("should trigger updating a credential when secret propagation change occurs.", func() {
		var err error

		secretID := "some-secret"
		password := "fakepassword"
		user := "fakeuser"

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
			Expect(os.Mkdir(dir, 0755)).To(Succeed())
			Expect(manager.watcher.Add(dir)).To(Succeed())
		}

		// Write the file
		Expect(os.WriteFile(filepath.Join(secretDirs[0], user), []byte(password), 0644)).To(Succeed())

		// set the expected command
		base64Str := base64.StdEncoding.EncodeToString([]byte(password))
		cmdSetPassword := fmt.Sprintf(`{"execute":"guest-set-user-password", "arguments": {"username":"%s", "password": "%s", "crypted": false }}`, user, base64Str)

		cmdPing := `{"execute":"guest-ping"}`
		mockConn.EXPECT().QemuAgentCommand(cmdPing, domName).AnyTimes().Return("", nil)

		domainSpec := expectIsolationDetectionForVMI(vmi)
		xml, err := xml.MarshalIndent(domainSpec, "", "\t")
		Expect(err).NotTo(HaveOccurred())

		mockDomain.EXPECT().Free().AnyTimes()
		mockConn.EXPECT().LookupDomainByName(domName).AnyTimes().Return(mockDomain, nil)
		mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
		mockDomain.EXPECT().GetXMLDesc(gomock.Any()).AnyTimes().Return(string(xml), nil)

		mockConn.EXPECT().DomainDefineXML(gomock.Any()).AnyTimes().DoAndReturn(func(xml string) (cli.VirDomain, error) {

			match := `			<accessCredential>
				<succeeded>true</succeeded>
			</accessCredential>`
			Expect(strings.Contains(xml, match)).To(BeTrue())
			return mockDomain, nil
		})

		matched := false
		mockConn.EXPECT().QemuAgentCommand(cmdSetPassword, domName).MinTimes(1).DoAndReturn(func(funcCmd string, funcDomName string) (string, error) {
			if funcCmd == cmdSetPassword {
				matched = true
			}
			return "", nil
		})

		// and wait
		go func() {
			watchTimeout := time.NewTicker(2 * time.Second)
			defer watchTimeout.Stop()
			<-watchTimeout.C
			close(manager.stopCh)
		}()

		// TODO: Rewrite test to not call private functions.
		manager.watchSecrets(vmi)
		Expect(matched).To(BeTrue())

		// And wait again after modifying file
		// Another execute command should occur with the updated password
		matched = false
		manager.stopCh = make(chan struct{})
		password = password + "morefake"
		Expect(os.WriteFile(filepath.Join(secretDirs[0], user), []byte(password), 0644)).To(Succeed())
		base64Str = base64.StdEncoding.EncodeToString([]byte(password))
		cmdSetPassword = fmt.Sprintf(`{"execute":"guest-set-user-password", "arguments": {"username":"%s", "password": "%s", "crypted": false }}`, user, base64Str)
		mockConn.EXPECT().QemuAgentCommand(cmdSetPassword, domName).MinTimes(1).Return("", nil)

		go func() {
			watchTimeout := time.NewTicker(2 * time.Second)
			defer watchTimeout.Stop()
			<-watchTimeout.C
			close(manager.stopCh)
		}()

		manager.watchSecrets(vmi)
	})

})
