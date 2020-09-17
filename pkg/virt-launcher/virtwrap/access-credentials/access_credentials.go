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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type openReturn struct {
	Return int `json:"return"`
}

type readReturnData struct {
	Count  int    `json:"count"`
	BufB64 string `json:"buf-b64"`
}
type readReturn struct {
	Return readReturnData `json:"return"`
}

type execReturn struct {
	Return execReturnData `json:"return"`
}
type execReturnData struct {
	Pid int `json:"pid"`
}

type execStatusReturn struct {
	Return execStatusReturnData `json:"return"`
}
type execStatusReturnData struct {
	Exited   bool   `json:"exited"`
	ExitCode int    `json:"exitcode"`
	OutData  string `json:"out-data"`
}

type AccessCredentialManager struct {
	virConn cli.Connection

	// access credentail propagation lock
	lock                 sync.Mutex
	secretWatcherStarted bool

	watcher *fsnotify.Watcher
}

func NewManager(connection cli.Connection) *AccessCredentialManager {

	return &AccessCredentialManager{
		virConn: connection,
	}
}

// only set during unit tests
var unitTestSecretDir string

func getSecretBaseDir() string {

	if unitTestSecretDir != "" {
		return unitTestSecretDir
	}

	return config.SecretSourceDir

}

func (l *AccessCredentialManager) writeGuestFile(contents string, domName string, filePath string) error {

	// ensure the directory exists with the correct permissions
	err := l.agentCreateDirectory(domName, filepath.Dir(filePath), "700")
	if err != nil {
		return err
	}

	// write the file
	base64Str := base64.StdEncoding.EncodeToString([]byte(contents))
	cmdOpenFile := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s", "mode":"w" } }`, filePath)
	output, err := l.virConn.QemuAgentCommand(cmdOpenFile, domName)
	if err != nil {
		return err
	}

	openRes := &openReturn{}
	err = json.Unmarshal([]byte(output), openRes)
	if err != nil {
		return err
	}

	cmdWriteFile := fmt.Sprintf(`{"execute": "guest-file-write", "arguments": { "handle": %d, "buf-b64": "%s" } }`, openRes.Return, base64Str)
	output, err = l.virConn.QemuAgentCommand(cmdWriteFile, domName)
	if err != nil {
		return err
	}

	cmdCloseFile := fmt.Sprintf(`{"execute": "guest-file-close", "arguments": { "handle": %d } }`, openRes.Return)
	output, err = l.virConn.QemuAgentCommand(cmdCloseFile, domName)
	if err != nil {
		return err
	}

	// ensure the file has the correct permissions and ownership
	l.agentSetFilePermissions(domName, filePath, "600")

	return nil
}

func (l *AccessCredentialManager) readGuestFile(domName string, filePath string) (string, error) {
	contents := ""

	cmdOpenFile := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": "%s", "mode":"r" } }`, filePath)
	output, err := l.virConn.QemuAgentCommand(cmdOpenFile, domName)
	if err != nil {
		return contents, err
	}

	openRes := &openReturn{}
	err = json.Unmarshal([]byte(output), openRes)
	if err != nil {
		return contents, err
	}

	cmdReadFile := fmt.Sprintf(`{"execute": "guest-file-read", "arguments": { "handle": %d } }`, openRes.Return)
	readOutput, err := l.virConn.QemuAgentCommand(cmdReadFile, domName)

	if err != nil {
		return contents, err
	}

	readRes := &readReturn{}
	err = json.Unmarshal([]byte(readOutput), readRes)
	if err != nil {
		return contents, err
	}

	if readRes.Return.Count > 0 {
		readBytes, err := base64.StdEncoding.DecodeString(readRes.Return.BufB64)
		if err != nil {
			return contents, err
		}
		contents = string(readBytes)
	}

	cmdCloseFile := fmt.Sprintf(`{"execute": "guest-file-close", "arguments": { "handle": %d } }`, openRes.Return)
	output, err = l.virConn.QemuAgentCommand(cmdCloseFile, domName)
	if err != nil {
		return contents, err
	}

	return contents, nil
}

func (l *AccessCredentialManager) agentGuestExec(domName string, command string, args []string) (string, error) {
	stdOut := ""
	argsStr := ""
	for _, arg := range args {
		if argsStr == "" {
			argsStr = fmt.Sprintf("\"%s\"", arg)
		} else {
			argsStr = argsStr + fmt.Sprintf(", \"%s\"", arg)
		}
	}

	cmdExec := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "%s", "arg": [ %s ], "capture-output":true } }`, command, argsStr)
	output, err := l.virConn.QemuAgentCommand(cmdExec, domName)
	execRes := &execReturn{}
	err = json.Unmarshal([]byte(output), execRes)
	if err != nil {
		return "", err
	}

	if execRes.Return.Pid <= 0 {
		return "", fmt.Errorf("Invalid pid [%d] returned from qemu agent during access credential injection: %s", execRes.Return.Pid, output)
	}

	exited := false
	exitCode := 0
	for i := 10; i > 0; i-- {
		cmdExecStatus := fmt.Sprintf(`{"execute": "guest-exec-status", "arguments": { "pid": %d } }`, execRes.Return.Pid)
		output, err := l.virConn.QemuAgentCommand(cmdExecStatus, domName)
		execStatusRes := &execStatusReturn{}
		err = json.Unmarshal([]byte(output), execStatusRes)
		if err != nil {
			return "", err
		}

		if execStatusRes.Return.Exited {
			stdOutBytes, err := base64.StdEncoding.DecodeString(execStatusRes.Return.OutData)
			if err != nil {
				return "", err
			}
			stdOut = string(stdOutBytes)
			exitCode = execStatusRes.Return.ExitCode
			exited = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !exited {
		return "", fmt.Errorf("Timed out waiting for guest pid [%d] for command [%s] to exit", execRes.Return.Pid, command)
	} else if exitCode != 0 {
		return stdOut, fmt.Errorf("Non-zero exit code [%d] for guest command [%s] with args [%v]: %s", exitCode, command, args, stdOut)
	}

	return stdOut, nil
}

// Requires usage of mkdir, chown, chmod, stat
func (l *AccessCredentialManager) agentCreateDirectory(domName string, dir string, permissions string) error {
	// Get parent directory ownership and use that for nested directory
	parentDir := filepath.Dir(dir)
	ownerStr, err := l.agentGetFileOwnership(domName, parentDir)
	if err != nil {
		return err
	}

	// Ensure the directory exists
	_, err = l.agentGuestExec(domName, "mkdir", []string{"-p", dir})
	if err != nil {
		return err
	}

	// set ownership/permissions of directory using parent directory owner
	_, err = l.agentGuestExec(domName, "chown", []string{ownerStr, dir})
	if err != nil {
		return err
	}
	_, err = l.agentGuestExec(domName, "chmod", []string{permissions, dir})
	if err != nil {
		return err
	}

	return nil
}

func (l *AccessCredentialManager) agentGetFileOwnership(domName string, filePath string) (string, error) {
	ownerStr, err := l.agentGuestExec(domName, "stat", []string{"-c", "%U:%G", filePath})
	if err != nil {
		return "", fmt.Errorf("Unable to detect ownership of access credential at %s: %s", filePath, err.Error())
	}
	ownerStr = strings.TrimSpace(ownerStr)
	if ownerStr == "" {
		return "", fmt.Errorf("Unable to detect ownership of access credential at %s", filePath)
	}

	log.Log.Infof("Detected owner %s for quest path %s", ownerStr, filePath)
	return ownerStr, nil
}

// Requires usage of chown, chmod, stat
func (l *AccessCredentialManager) agentSetFilePermissions(domName string, filePath string, permissions string) error {
	// Get parent directory ownership and use that for nested directory
	parentDir := filepath.Dir(filePath)
	ownerStr, err := l.agentGetFileOwnership(domName, parentDir)
	if err != nil {
		return err
	}

	// set ownership/permissions of directory using parent directory owner
	_, err = l.agentGuestExec(domName, "chown", []string{ownerStr, filePath})
	if err != nil {
		return err
	}
	_, err = l.agentGuestExec(domName, "chmod", []string{permissions, filePath})
	if err != nil {
		return err
	}

	return nil
}

func (l *AccessCredentialManager) agentSetUserPassword(domName string, user string, password string) error {

	base64Str := base64.StdEncoding.EncodeToString([]byte(password))

	cmdSetPassword := fmt.Sprintf(`{"execute":"guest-set-user-password", "arguments": {"username":"%s", "password": "%s", "crypted": false }}`, user, base64Str)

	_, err := l.virConn.QemuAgentCommand(cmdSetPassword, domName)
	if err != nil {
		return err
	}
	return nil
}

func (l *AccessCredentialManager) agentWriteAuthorizedKeys(domName string, filePath string, authorizedKeys string) error {

	separator := "\n### AUTO PROPAGATED BY KUBEVIRT BELOW THIS LINE ###\n"
	curAuthorizedKeys := ""

	// ######
	// Step 1. Read file on guest
	// ######
	curAuthorizedKeys, err := l.readGuestFile(domName, filePath)
	if err != nil && !strings.Contains(err.Error(), "No such file or directory") {
		return err
	}

	// ######
	// Step 2. Merge kubevirt authorized keys to end of file
	// ######

	// Add a warning line so people know where these entries are coming from
	// and the risk of altering them
	origAuthorizedKeys := curAuthorizedKeys
	split := strings.Split(curAuthorizedKeys, separator)
	if len(split) > 0 {
		curAuthorizedKeys = split[0]
	} else {
		curAuthorizedKeys = ""
	}
	authorizedKeys = fmt.Sprintf("%s%s%s", curAuthorizedKeys, separator, authorizedKeys)

	// ######
	// Step 3. Write merged file
	// ######
	// only update if the updated string is not equal to the current contents on the guest.
	if origAuthorizedKeys != authorizedKeys {
		err = l.writeGuestFile(authorizedKeys, domName, filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func isSSHPublicKey(accessCred *v1.AccessCredential) bool {
	if accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
		return true
	}

	return false
}

func isUserPassword(accessCred *v1.AccessCredential) bool {

	if accessCred.UserPassword != nil && accessCred.UserPassword.PropagationMethod.QemuGuestAgent != nil {
		return true
	}

	return false
}

func getSecret(accessCred *v1.AccessCredential) string {
	secretName := ""
	if accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
		if accessCred.SSHPublicKey.Source.Secret != nil {
			secretName = accessCred.SSHPublicKey.Source.Secret.SecretName
		}
	} else if accessCred.UserPassword != nil && accessCred.UserPassword.PropagationMethod.QemuGuestAgent != nil {
		if accessCred.UserPassword.Source.Secret != nil {
			secretName = accessCred.UserPassword.Source.Secret.SecretName
		}
	}

	return secretName
}

func (l *AccessCredentialManager) watchSecrets(vmi *v1.VirtualMachineInstance) {
	logger := log.Log.Object(vmi)

	reload := true
	fileChangeDetected := true

	domName := util.VMINamespaceKeyFunc(vmi)

	// guest agent will force a resync of changes every 'x' minutes
	forceResyncTicker := time.NewTicker(5 * time.Minute)
	defer forceResyncTicker.Stop()

	// guest agent will aggregate all changes to secrets and apply them
	// every 'x' seconds. This could help prevent making multiple qemu
	// execution calls in the event multiple secret changes are landing within
	// a small timeframe.
	handleChangesTicker := time.NewTicker(15 * time.Second)
	defer handleChangesTicker.Stop()

	for {
		select {
		case <-l.watcher.Events:
			fileChangeDetected = true
		case err := <-l.watcher.Errors:
			logger.Reason(err).Errorf("Error encountered while watching downward api secret")
		case <-forceResyncTicker.C:
			reload = true
			logger.Info("Resyncing access credentials due to recurring resync period")
		case <-handleChangesTicker.C:
			if fileChangeDetected {
				reload = true
				logger.Info("Reloading access credentials because secret changed")
			}
		}

		if !reload {
			continue
		}

		fileChangeDetected = false
		reload = false

		// secret name mapped to authorized_keys in that secret
		secretMap := make(map[string]string)
		// filepath mapped to secretNames
		filePathMap := make(map[string][]string)
		// maps users to passwords
		userPasswordMap := make(map[string]string)

		// Step 1. Populate Secrets and filepath Map
		for _, accessCred := range vmi.Spec.AccessCredentials {
			secretName := getSecret(&accessCred)
			if secretName == "" {
				continue
			}

			secretDir := filepath.Join(getSecretBaseDir(), secretName+"-access-cred")
			files, err := ioutil.ReadDir(secretDir)
			if err != nil {
				// if reading failed, reset reload to true so this change will be retried again
				reload = true
				logger.Reason(err).Errorf("Error encountered reading secrets file list from base directory %s", secretDir)
				continue
			}

			if isSSHPublicKey(&accessCred) {
				for _, entry := range accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent.AuthorizedKeysFiles {
					secrets, ok := filePathMap[entry.FilePath]
					if !ok {
						filePathMap[entry.FilePath] = []string{secretName}
					} else {
						filePathMap[entry.FilePath] = append(secrets, secretName)
					}
				}

				authorizedKeys := ""
				for _, file := range files {
					if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
						continue
					}

					pubKeyBytes, err := ioutil.ReadFile(filepath.Join(secretDir, file.Name()))
					if err != nil {
						// if reading failed, reset reload to true so this change will be retried again
						reload = true
						logger.Reason(err).Errorf("Error encountered reading secret file %s", filepath.Join(secretDir, file.Name()))
						continue
					}

					pubKey := string(pubKeyBytes)
					if pubKey == "" {
						continue
					}
					authorizedKeys = fmt.Sprintf("%s\n%s", authorizedKeys, pubKey)
				}

				secretMap[secretName] = authorizedKeys
			} else if isUserPassword(&accessCred) {
				for _, file := range files {
					if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
						continue
					}

					passwordBytes, err := ioutil.ReadFile(filepath.Join(secretDir, file.Name()))
					if err != nil {
						// if reading failed, reset reload to true so this change will be retried again
						reload = true
						logger.Reason(err).Errorf("Error encountered reading secret file %s", filepath.Join(secretDir, file.Name()))
						continue
					}

					password := strings.TrimSpace(string(passwordBytes))
					if password == "" {
						continue
					}
					userPasswordMap[file.Name()] = password
				}
			}
		}

		// Step 2. Update Authorized keys file
		for filePath, secretNames := range filePathMap {
			authorizedKeys := ""

			for _, secretName := range secretNames {
				pubKeys, ok := secretMap[secretName]
				if ok && pubKeys != "" {
					authorizedKeys = fmt.Sprintf("%s\n%s", authorizedKeys, pubKeys)
				}
			}

			err := l.agentWriteAuthorizedKeys(domName, filePath, authorizedKeys)
			if err != nil {
				// if writing failed, reset reload to true so this change will be retried again
				reload = true
				logger.Reason(err).Errorf("Error encountered writing access credentials using guest agent")
				continue
			}
		}

		// Step 3. update UserPasswords
		for user, password := range userPasswordMap {
			err := l.agentSetUserPassword(domName, user, password)
			if err != nil {
				// if setting password failed, reset reload to true so this will be tried again
				reload = true
				logger.Reason(err).Errorf("Error encountered setting password for user [%s]", user)
				continue
			}
		}
	}
}

func (l *AccessCredentialManager) HandleQemuAgentAccessCredentials(vmi *v1.VirtualMachineInstance) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.secretWatcherStarted {
		// already started
		return nil
	}

	found := false
	for _, accessCred := range vmi.Spec.AccessCredentials {
		if accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
			found = true
			break
		} else if accessCred.UserPassword != nil && accessCred.UserPassword.PropagationMethod.QemuGuestAgent != nil {
			found = true
			break
		}
	}

	if !found {
		// not using the agent for ssh pub key propagation
		return nil
	}

	var err error
	l.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, accessCred := range vmi.Spec.AccessCredentials {
		secretName := ""
		if accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
			if accessCred.SSHPublicKey.Source.Secret != nil {
				secretName = accessCred.SSHPublicKey.Source.Secret.SecretName
			}
		} else if accessCred.UserPassword != nil && accessCred.UserPassword.PropagationMethod.QemuGuestAgent != nil {
			if accessCred.UserPassword.Source.Secret != nil {
				secretName = accessCred.UserPassword.Source.Secret.SecretName
			}
		}

		if secretName == "" {
			continue
		}
		secretDir := filepath.Join(getSecretBaseDir(), secretName+"-access-cred")
		err = l.watcher.Add(secretDir)
		if err != nil {
			return err
		}
	}

	go l.watchSecrets(vmi)
	l.secretWatcherStarted = true

	return nil
}
