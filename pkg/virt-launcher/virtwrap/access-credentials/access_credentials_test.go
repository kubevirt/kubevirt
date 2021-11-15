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
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
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

		manager = NewManager(mockConn, &lock)
		manager.resyncCheckIntervalSeconds = 1
		tmpDir, err = ioutil.TempDir("", "credential-test")
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

		res, err := manager.agentGuestExec(domName, command, args)
		Expect(err).To(BeNil())
		Expect(res).To(Equal("ssh somekey123 test-key\n"))
	})

	It("should handle dynamically updating user/password with qemu agent", func() {

		domName := "some-domain"
		password := "1234"
		user := "myuser"
		base64Str := base64.StdEncoding.EncodeToString([]byte(password))
		cmdSetPassword := fmt.Sprintf(`{"execute":"guest-set-user-password", "arguments": {"username":"%s", "password": "%s", "crypted": false }}`, user, base64Str)
		mockConn.EXPECT().QemuAgentCommand(cmdSetPassword, domName).Return("", nil)

		err := manager.agentSetUserPassword(domName, user, password)
		Expect(err).To(BeNil())
	})

	It("should handle dynamically updating ssh key with qemu agent", func() {
		domName := "some-domain"
		user := "someowner"
		filePath := "/home/someowner/.ssh"

		authorizedKeys := "ssh some injected key"

		expectedOpenCmd := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s/authorized_keys", "mode":"r" } }`, filePath)
		expectedWriteOpenCmd := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s/authorized_keys", "mode":"w" } }`, filePath)
		expectedOpenCmdRes := `{"return":1000}`

		existingKey := base64.StdEncoding.EncodeToString([]byte("ssh some existing key"))
		expectedReadCmd := `{"execute": "guest-file-read", "arguments": { "handle": 1000 } }`
		expectedReadCmdRes := fmt.Sprintf(`{"return":{"count":24,"buf-b64": "%s"}}`, existingKey)

		mergedKeys := base64.StdEncoding.EncodeToString([]byte(authorizedKeys))
		expectedWriteCmd := fmt.Sprintf(`{"execute": "guest-file-write", "arguments": { "handle": 1000, "buf-b64": "%s" } }`, mergedKeys)

		expectedCloseCmd := `{"execute": "guest-file-close", "arguments": { "handle": 1000 } }`

		expectedExecReturn := `{"return":{"pid":789}}`
		expectedStatusCmd := `{"execute": "guest-exec-status", "arguments": { "pid": 789 } }`

		getentBase64Str := base64.StdEncoding.EncodeToString([]byte("someowner:x:1111:2222:Some Owner:/home/someowner:/bin/bash"))
		expectedHomeDirCmd := `{"execute": "guest-exec", "arguments": { "path": "getent", "arg": [ "passwd", "someowner" ], "capture-output":true } }`
		expectedHomeDirCmdRes := fmt.Sprintf(`{"return":{"exitcode":0,"out-data":"%s","exited":true}}`, getentBase64Str)

		expectedMkdirCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "mkdir", "arg": [ "-p", "%s" ], "capture-output":true } }`, filePath)
		expectedMkdirRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedParentChownCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chown", "arg": [ "1111:2222", "%s" ], "capture-output":true } }`, filePath)
		expectedParentChownRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedParentChmodCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chmod", "arg": [ "700", "%s" ], "capture-output":true } }`, filePath)
		expectedParentChmodRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedFileChownCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chown", "arg": [ "1111:2222", "%s/authorized_keys" ], "capture-output":true } }`, filePath)
		expectedFileChownRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedFileChmodCmd := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "chmod", "arg": [ "600", "%s/authorized_keys" ], "capture-output":true } }`, filePath)
		expectedFileChmodRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		//
		//
		//
		//
		// Detect user home dir
		//
		mockConn.EXPECT().QemuAgentCommand(expectedHomeDirCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedHomeDirCmdRes, nil)

		//
		//
		//
		// Expected Read File
		//
		mockConn.EXPECT().QemuAgentCommand(expectedOpenCmd, domName).Return(expectedOpenCmdRes, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedReadCmd, domName).Return(expectedReadCmdRes, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedCloseCmd, domName).Return("", nil)

		//
		//
		//
		// Expected prepare directory
		//
		mockConn.EXPECT().QemuAgentCommand(expectedMkdirCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedMkdirRes, nil)

		mockConn.EXPECT().QemuAgentCommand(expectedParentChownCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedParentChownRes, nil)

		mockConn.EXPECT().QemuAgentCommand(expectedParentChmodCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedParentChmodRes, nil)

		//
		//
		//
		// Expected Write file
		//
		mockConn.EXPECT().QemuAgentCommand(expectedWriteOpenCmd, domName).Return(expectedOpenCmdRes, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedWriteCmd, domName).Return("", nil)
		mockConn.EXPECT().QemuAgentCommand(expectedCloseCmd, domName).Return("", nil)

		//
		//
		//
		// Expected set file permissions
		//
		mockConn.EXPECT().QemuAgentCommand(expectedFileChownCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedFileChownRes, nil)

		mockConn.EXPECT().QemuAgentCommand(expectedFileChmodCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedFileChmodRes, nil)

		err := manager.agentWriteAuthorizedKeys(domName, user, authorizedKeys)
		Expect(err).To(BeNil())
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

		manager.watcher, err = fsnotify.NewWatcher()
		Expect(err).To(BeNil())

		secretDirs := getSecretDirs(vmi)
		Expect(len(secretDirs)).To(Equal(1))
		Expect(secretDirs[0]).To(Equal(fmt.Sprintf("%s/%s-access-cred", tmpDir, secretID)))

		for _, dir := range secretDirs {
			os.Mkdir(dir, 0755)
			err = manager.watcher.Add(dir)
			Expect(err).To(BeNil())
		}

		// Write the file
		err = ioutil.WriteFile(secretDirs[0]+"/"+user, []byte(password), 0644)
		Expect(err).To(BeNil())

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

		manager.watchSecrets(vmi)
		Expect(matched).To(Equal(true))

		// And wait again after modifying file
		// Another execute command should occur with the updated password
		matched = false
		manager.stopCh = make(chan struct{})
		password = password + "morefake"
		err = ioutil.WriteFile(secretDirs[0]+"/"+user, []byte(password), 0644)
		Expect(err).To(BeNil())
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
