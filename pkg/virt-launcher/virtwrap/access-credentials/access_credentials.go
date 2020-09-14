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

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type openReturn struct {
	Return int `json:"return"`
}
type readReturn struct {
	Count  int    `json:"count"`
	BufB64 string `json:"buf-b64"`
}

type AccessCredentialManager struct {
	virConn cli.Connection

	// access credentail propagation lock
	lock                 sync.Mutex
	secretWatcherStarted bool
}

func NewManager(connection cli.Connection) *AccessCredentialManager {

	return &AccessCredentialManager{
		virConn: connection,
	}
}

func (l *AccessCredentialManager) writeGuestFile(contents string, domName string, filePath string) error {

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

	log.Log.Infof("VOSSEL -DEBUG- READOUTPUT: %s", readOutput)
	if err != nil {
		return contents, err
	}

	readRes := &readReturn{}
	err = json.Unmarshal([]byte(readOutput), readRes)
	if err != nil {
		return contents, err
	}

	if readRes.Count > 0 {
		readBytes, err := base64.StdEncoding.DecodeString(readRes.BufB64)
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

func (l *AccessCredentialManager) agentWriteAuthorizedKeys(domName string, filePath string, authorizedKeys string) error {

	separator := "### AUTO PROPAGATED BY KUBEVIRT BELOW THIS LINE ###\n"
	curAuthorizedKeys := ""

	// ######
	// Step 1. Read file on guest
	// ######

	curAuthorizedKeys, err := l.readGuestFile(domName, filePath)
	if err != nil {
		if strings.Contains(err.Error(), "No such file or directory") {
			err = fmt.Errorf("Unable to update authorized_keys file because file does not exist on the guest at path %s. Error: %v", filePath, err.Error())
		}

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
	authorizedKeys = fmt.Sprintf("%s\n%s\n%s", curAuthorizedKeys, separator, authorizedKeys)

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

func (l *AccessCredentialManager) watchSecrets(vmi *v1.VirtualMachineInstance) {
	logger := log.Log.Object(vmi)

	domName := util.VMINamespaceKeyFunc(vmi)
	for {
		// secret name mapped to authorized_keys in that secret
		secretMap := make(map[string]string)
		// filepath mapped to secretNames
		filePathMap := make(map[string][]string)

		// TODO make this inotify based
		time.Sleep(10 * time.Second)

		// Step 1. Populate Secrets and filepath Map
		for _, accessCred := range vmi.Spec.AccessCredentials {
			if accessCred.SSHPublicKey == nil || accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent == nil {
				continue
			}

			secretName := ""
			if accessCred.SSHPublicKey.Source.Secret != nil {
				secretName = accessCred.SSHPublicKey.Source.Secret.SecretName
			}

			if secretName == "" {
				continue
			}

			for _, entry := range accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent.AuthorizedKeysFiles {
				secrets, ok := filePathMap[entry.FilePath]
				if !ok {
					filePathMap[entry.FilePath] = []string{secretName}
				} else {
					filePathMap[entry.FilePath] = append(secrets, secretName)
				}
			}

			secretDir := filepath.Join(config.SecretSourceDir, secretName+"-access-cred")
			files, err := ioutil.ReadDir(secretDir)
			if err != nil {
				logger.Reason(err).Errorf("Error encountered reading secrets file list from base directory %s", secretDir)
				continue
			}

			authorizedKeys := ""
			for _, file := range files {
				if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
					continue
				}

				pubKeyBytes, err := ioutil.ReadFile(filepath.Join(secretDir, file.Name()))
				if err != nil {
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
				logger.Reason(err).Errorf("Error encountered writing access credentials using guest agent")
				continue
			}
		}
	}
}

func (l *AccessCredentialManager) HandleQemuAgentAccessCredentials(vmi *v1.VirtualMachineInstance) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.secretWatcherStarted {
		// already started
		return
	}

	found := false
	for _, accessCred := range vmi.Spec.AccessCredentials {
		if accessCred.SSHPublicKey == nil || accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent == nil {
			continue
		}
		found = true
		break
	}

	if !found {
		// not using the agent for ssh pub key propagation
		return
	}

	go l.watchSecrets(vmi)
	l.secretWatcherStarted = true
}
