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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"

	"github.com/fsnotify/fsnotify"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
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

type AccessCredentialManager struct {
	virConn cli.Connection

	// access credential propagation watchLock
	watchLock            sync.Mutex
	secretWatcherStarted bool

	stopCh                     chan struct{}
	doneCh                     chan struct{}
	resyncCheckIntervalSeconds int

	watcher *fsnotify.Watcher

	domainModifyLock *sync.Mutex
	metadataCache    *metadata.Cache
}

func NewManager(connection cli.Connection, domainModifyLock *sync.Mutex, metadataCache *metadata.Cache) *AccessCredentialManager {
	return &AccessCredentialManager{
		virConn:                    connection,
		resyncCheckIntervalSeconds: 15,
		domainModifyLock:           domainModifyLock,
		metadataCache:              metadataCache,
	}
}

// only set during unit tests
var unitTestSecretDir string

func getSecretDirs(vmi *v1.VirtualMachineInstance) []string {
	var dirs []string
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
		dirs = append(dirs, getSecretDir(secretName))
	}

	return dirs
}

func getSecretDir(secretName string) string {
	return filepath.Join(getSecretBaseDir(), secretName+"-access-cred")
}

func getSecretBaseDir() string {

	if unitTestSecretDir != "" {
		return unitTestSecretDir
	}

	return config.SecretSourceDir

}

func (l *AccessCredentialManager) writeGuestFile(contents string, domName string, filePath string, owner string, fileExists bool) error {

	// ensure the directory exists with the correct permissions
	err := l.agentCreateDirectory(domName, filepath.Dir(filePath), "700", owner)
	if err != nil {
		return err
	}

	if fileExists {
		// ensure the file has the correct permissions for writing
		l.agentSetFilePermissions(domName, filePath, "600", owner)

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
	_, err = l.virConn.QemuAgentCommand(cmdWriteFile, domName)
	if err != nil {
		return err
	}

	cmdCloseFile := fmt.Sprintf(`{"execute": "guest-file-close", "arguments": { "handle": %d } }`, openRes.Return)
	_, err = l.virConn.QemuAgentCommand(cmdCloseFile, domName)
	if err != nil {
		return err
	}

	if !fileExists {
		// ensure the file has the correct permissions and ownership after creating new file
		l.agentSetFilePermissions(domName, filePath, "600", owner)
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
	_, err = l.virConn.QemuAgentCommand(cmdCloseFile, domName)
	if err != nil {
		return contents, err
	}

	return contents, nil
}

func (l *AccessCredentialManager) agentGuestExec(domName string, command string, args []string) (string, error) {
	return agent.GuestExec(l.virConn, domName, command, args, 10)
}

// Requires usage of mkdir, chown, chmod
func (l *AccessCredentialManager) agentCreateDirectory(domName string, dir string, permissions string, owner string) error {
	// Ensure the directory exists
	_, err := l.agentGuestExec(domName, "mkdir", []string{"-p", dir})
	if err != nil {
		return err
	}

	// set ownership/permissions of directory using parent directory owner
	_, err = l.agentGuestExec(domName, "chown", []string{owner, dir})
	if err != nil {
		return err
	}
	_, err = l.agentGuestExec(domName, "chmod", []string{permissions, dir})
	if err != nil {
		return err
	}

	return nil
}

func (l *AccessCredentialManager) agentGetUserInfo(domName string, user string) (string, string, string, error) {
	passwdEntryStr, err := l.agentGuestExec(domName, "getent", []string{"passwd", user})
	if err != nil {
		return "", "", "", fmt.Errorf("Unable to detect home directory of user %s: %s", user, err.Error())
	}
	passwdEntryStr = strings.TrimSpace(passwdEntryStr)
	entries := strings.Split(passwdEntryStr, ":")
	if len(entries) < 6 {
		return "", "", "", fmt.Errorf("Unable to detect home directory of user %s", user)
	}

	filePath := entries[5]
	uid := entries[2]
	gid := entries[3]
	log.Log.Infof("Detected home directory %s for user %s", filePath, user)
	return filePath, uid, gid, nil
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

// Requires usage of chown, chmod
func (l *AccessCredentialManager) agentSetFilePermissions(domName string, filePath string, permissions string, owner string) error {
	// set ownership/permissions of directory using parent directory owner
	_, err := l.agentGuestExec(domName, "chown", []string{owner, filePath})
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

func (l *AccessCredentialManager) pingAgent(domName string) error {
	cmdPing := `{"execute":"guest-ping"}`

	_, err := l.virConn.QemuAgentCommand(cmdPing, domName)
	return err
}

func (l *AccessCredentialManager) agentSetAuthorizedKeys(domName string, user string, authorizedKeys []string) error {
	err := func() error {
		domain, err := l.virConn.LookupDomainByName(domName)
		if err != nil {
			return err
		}
		defer domain.Free()

		// Zero flags argument means that the authorized_keys file is overwritten with the authorizedKeys
		return domain.AuthorizedSSHKeysSet(user, authorizedKeys, 0)
	}()
	if errors.Is(err, libvirt.ERR_NO_SUPPORT) {
		// If AuthorizedSSHKeysSet method is not supported, use the old method
		desiredAuthorizedKeys := strings.Join(authorizedKeys, "\n")
		return l.agentWriteAuthorizedKeysFile(domName, user, desiredAuthorizedKeys)
	}
	return err
}

func (l *AccessCredentialManager) agentWriteAuthorizedKeysFile(domName string, user string, desiredAuthorizedKeys string) (err error) {
	curAuthorizedKeys := ""
	fileExists := true

	// ######
	// Step 1. Get home directory for user.
	// ######
	homeDir, uid, gid, err := l.agentGetUserInfo(domName, user)
	if err != nil {
		return err
	}

	// ######
	// Step 2. Read file on guest to determine if change is required
	// ######
	filePath := fmt.Sprintf("%s/.ssh/authorized_keys", homeDir)
	curAuthorizedKeys, err = l.readGuestFile(domName, filePath)
	if err != nil && strings.Contains(err.Error(), "No such file or directory") {
		fileExists = false
	} else if err != nil {
		return err
	}

	// ######
	// Step 3. Write authorized_keys file if changes exist
	// ######
	// only update if the updated string is not equal to the current contents on the guest.
	if curAuthorizedKeys != desiredAuthorizedKeys {
		err = l.writeGuestFile(desiredAuthorizedKeys, domName, filePath, fmt.Sprintf("%s:%s", uid, gid), fileExists)
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

func (l *AccessCredentialManager) reportAccessCredentialResult(succeeded bool, message string) {
	acMetadata := api.AccessCredentialMetadata{
		Succeeded: succeeded,
		Message:   message,
	}
	l.metadataCache.AccessCredential.Store(acMetadata)
	log.Log.V(4).Infof("Access credential set in metadata: %v", acMetadata)
	return
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
	handleChangesTicker := time.NewTicker(time.Duration(l.resyncCheckIntervalSeconds) * time.Second)
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
		case <-l.stopCh:
			logger.Info("Signalled to stop watching access credential secrets")
			return
		}

		if !reload {
			continue
		}

		fileChangeDetected = false
		reload = false
		reportedErr := false

		err := l.pingAgent(domName)
		if err != nil {
			reload = true
			reportedErr = true
			l.reportAccessCredentialResult(false, "Guest agent is offline")
			continue
		}

		credentialInfo := newAccessCredentialsInfo()

		// Step 1. Populate access credential info
		for i := range vmi.Spec.AccessCredentials {
			err := credentialInfo.addAccessCredential(&vmi.Spec.AccessCredentials[i])
			if err != nil {
				// if reading failed, reset reload to true so this change will be retried again
				reload = true
				reportedErr = true
				logger.Reason(err).Errorf("Error encountered")
				l.reportAccessCredentialResult(false, err.Error())
			}
		}

		// Step 2. Update Authorized keys
		for user, secretNames := range credentialInfo.userSSHMap {
			var allAuthorizedKeys []string
			for _, secretName := range secretNames {
				pubKeys := credentialInfo.secretMap[secretName]
				allAuthorizedKeys = append(allAuthorizedKeys, pubKeys...)
			}

			err := l.agentSetAuthorizedKeys(domName, user, allAuthorizedKeys)
			if err != nil {
				// if writing failed, reset reload to true so this change will be retried again
				reload = true
				reportedErr = true
				logger.Reason(err).Errorf("Error encountered writing access credentials using guest agent")
				l.reportAccessCredentialResult(false, fmt.Sprintf("Error encountered writing ssh pub key access credentials for user [%s]: %v", user, err))
				continue
			}
		}

		// Step 3. update UserPasswords
		for user, password := range credentialInfo.userPasswordMap {
			err := l.agentSetUserPassword(domName, user, password)
			if err != nil {
				// if setting password failed, reset reload to true so this will be tried again
				reload = true
				reportedErr = true
				logger.Reason(err).Errorf("Error encountered setting password for user [%s]", user)

				l.reportAccessCredentialResult(false, fmt.Sprintf("Error encountered setting password for user [%s]: %v", user, err))
				continue
			}
		}
		if !reportedErr {
			l.reportAccessCredentialResult(true, "")
		}
	}
}

func (l *AccessCredentialManager) HandleQemuAgentAccessCredentials(vmi *v1.VirtualMachineInstance) error {
	l.watchLock.Lock()
	defer l.watchLock.Unlock()

	if l.secretWatcherStarted {
		// already started
		return nil
	}

	secretDirs := getSecretDirs(vmi)
	if len(secretDirs) == 0 {
		// nothing to watch
		return nil
	}

	var err error
	l.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, dir := range secretDirs {
		err = l.watcher.Add(dir)
		if err != nil {
			// Ignoring error returned by watcher.Close(),
			// because another error will be returned.
			_ = l.watcher.Close()
			return err
		}
	}

	l.stopCh = make(chan struct{})
	l.doneCh = make(chan struct{})

	go func() {
		defer close(l.doneCh)
		// Ignoring the error, because the watch has stopped.
		defer func() { _ = l.watcher.Close() }()
		l.watchSecrets(vmi)
	}()

	l.secretWatcherStarted = true

	return nil
}

func (l *AccessCredentialManager) Stop() {
	l.watchLock.Lock()
	defer l.watchLock.Unlock()

	if !l.secretWatcherStarted {
		return
	}

	close(l.stopCh)
	<-l.doneCh

	l.secretWatcherStarted = false
}

type accessCredentialsInfo struct {
	// secret name mapped to authorized_keys in that secret
	secretMap map[string][]string
	// filepath mapped to secretNames
	userSSHMap map[string][]string
	// maps users to passwords
	userPasswordMap map[string]string
}

func (a *accessCredentialsInfo) addAccessCredential(accessCred *v1.AccessCredential) error {
	secretName := getSecret(accessCred)
	if secretName == "" {
		return nil
	}

	secretDir := getSecretDir(secretName)
	files, err := os.ReadDir(secretDir)
	if err != nil {
		return fmt.Errorf("error occurred while reading the list of secrets files from the base directory %s: %w", secretDir, err)
	}

	if isSSHPublicKey(accessCred) {
		for _, user := range accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent.Users {
			a.userSSHMap[user] = append(a.userSSHMap[user], secretName)
		}

		var authorizedKeys []string
		for _, file := range files {
			if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
				continue
			}

			pubKeyBytes, err := os.ReadFile(filepath.Join(secretDir, file.Name()))
			if err != nil {
				return fmt.Errorf("error occurred while reading the access credential secret file [%s]: %w", filepath.Join(secretDir, file.Name()), err)
			}

			for _, pubKey := range strings.Split(string(pubKeyBytes), "\n") {
				trimmedKey := strings.TrimSpace(pubKey)
				if trimmedKey != "" {
					authorizedKeys = append(authorizedKeys, trimmedKey)
				}
			}
		}

		if len(authorizedKeys) > 0 {
			a.secretMap[secretName] = authorizedKeys
		}

	} else if isUserPassword(accessCred) {
		for _, file := range files {
			// Mounted secret directory contains directories prefixed by "..".
			// They are used by k8s to atomically swap all files when the secret is updated.
			if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
				continue
			}

			passwordBytes, err := os.ReadFile(filepath.Join(secretDir, file.Name()))
			if err != nil {
				return fmt.Errorf("error occurred while reading the access credential secret file [%s]: %w", filepath.Join(secretDir, file.Name()), err)
			}

			password := strings.TrimSpace(string(passwordBytes))
			if password == "" {
				continue
			}
			a.userPasswordMap[file.Name()] = password
		}
	}

	return nil
}

func newAccessCredentialsInfo() *accessCredentialsInfo {
	return &accessCredentialsInfo{
		secretMap:       make(map[string][]string),
		userSSHMap:      make(map[string][]string),
		userPasswordMap: make(map[string]string),
	}
}
