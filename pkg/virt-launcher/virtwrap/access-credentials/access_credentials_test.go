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
	"fmt"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/mock/gomock"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("AccessCredentials", func() {
	var mockConn *cli.MockConnection
	var ctrl *gomock.Controller
	var manager *AccessCredentialManager
	var tmpDir string

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		manager = NewManager(mockConn)
		manager.resyncCheckIntervalSeconds = 1
		tmpDir, _ = ioutil.TempDir("", "credential-test")
		unitTestSecretDir = tmpDir
	})
	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

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
		filePath := "/some/file/path/authorized_keys"

		authorizedKeys := "some injected ssh key"
		base64Str := "c3NoIHNvbWVrZXkgc3R1ZmYKCiMjIyBBVVRPIFBST1BBR0FURUQgQlkgS1VCRVZJUlQgQkVMT1cgVEhJUyBMSU5FICMjIwpzb21lIGluamVjdGVkIHNzaCBrZXk="

		expectedOpenCmd := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s", "mode":"r" } }`, filePath)
		expectedWriteOpenCmd := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s", "mode":"w" } }`, filePath)
		expectedOpenCmdRes := `{"return":1000}`

		expectedReadCmd := `{"execute": "guest-file-read", "arguments": { "handle": 1000 } }`
		expectedReadCmdRes := `{"return":{"count":24,"buf-b64": "c3NoIHNvbWVrZXkgc3R1ZmYK"}}`

		expectedWriteCmd := fmt.Sprintf(`{"execute": "guest-file-write", "arguments": { "handle": 1000, "buf-b64": "%s" } }`, base64Str)

		expectedCloseCmd := `{"execute": "guest-file-close", "arguments": { "handle": 1000 } }`

		expectedExecReturn := `{"return":{"pid":789}}`
		expectedStatusCmd := `{"execute": "guest-exec-status", "arguments": { "pid": 789 } }`

		expectedParentOwnerCmd := `{"execute": "guest-exec", "arguments": { "path": "stat", "arg": [ "-c", "%U:%G", "/some/file" ], "capture-output":true } }`
		expectedParentOwnerCmdRes := `{"return":{"exitcode":0,"out-data":"dXNlcjpwYXNz","exited":true}}`

		expectedFileOwnerCmd := `{"execute": "guest-exec", "arguments": { "path": "stat", "arg": [ "-c", "%U:%G", "/some/file/path" ], "capture-output":true } }`
		expectedFileOwnerCmdRes := `{"return":{"exitcode":0,"out-data":"dXNlcjpwYXNz","exited":true}}`

		expectedMkdirCmd := `{"execute": "guest-exec", "arguments": { "path": "mkdir", "arg": [ "-p", "/some/file/path" ], "capture-output":true } }`
		expectedMkdirRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedParentChownCmd := `{"execute": "guest-exec", "arguments": { "path": "chown", "arg": [ "user:pass", "/some/file/path" ], "capture-output":true } }`
		expectedParentChownRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedParentChmodCmd := `{"execute": "guest-exec", "arguments": { "path": "chmod", "arg": [ "700", "/some/file/path" ], "capture-output":true } }`
		expectedParentChmodRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedFileChownCmd := `{"execute": "guest-exec", "arguments": { "path": "chown", "arg": [ "user:pass", "/some/file/path/authorized_keys" ], "capture-output":true } }`
		expectedFileChownRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`

		expectedFileChmodCmd := `{"execute": "guest-exec", "arguments": { "path": "chmod", "arg": [ "600", "/some/file/path/authorized_keys" ], "capture-output":true } }`
		expectedFileChmodRes := `{"return":{"exitcode":0,"out-data":"","exited":true}}`
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
		mockConn.EXPECT().QemuAgentCommand(expectedParentOwnerCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedParentOwnerCmdRes, nil)

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

		mockConn.EXPECT().QemuAgentCommand(expectedFileOwnerCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedFileOwnerCmdRes, nil)

		mockConn.EXPECT().QemuAgentCommand(expectedFileChownCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedFileChownRes, nil)

		mockConn.EXPECT().QemuAgentCommand(expectedFileChmodCmd, domName).Return(expectedExecReturn, nil)
		mockConn.EXPECT().QemuAgentCommand(expectedStatusCmd, domName).Return(expectedFileChmodRes, nil)

		err := manager.agentWriteAuthorizedKeys(domName, filePath, authorizedKeys)
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
