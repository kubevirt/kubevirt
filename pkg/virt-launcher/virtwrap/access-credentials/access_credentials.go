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

	"github.com/fsnotify/fsnotify"
	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const logVerbosityDebug = 4

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

	eventSent   bool
	eventSender EventSender
}

type EventSender interface {
	SendK8sEvent(vmi *v1.VirtualMachineInstance, severity, reason, message string) error
}

func NewManager(connection cli.Connection,
	domainModifyLock *sync.Mutex,
	metadataCache *metadata.Cache,
	eventSender EventSender,
) *AccessCredentialManager {
	return &AccessCredentialManager{
		virConn:                    connection,
		resyncCheckIntervalSeconds: 15,
		domainModifyLock:           domainModifyLock,
		metadataCache:              metadataCache,
		eventSender:                eventSender,
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

func (l *AccessCredentialManager) writeGuestFile(contents, domName, filePath, owner string, fileExists bool) error {
	// ensure the directory exists with the correct permissions
	err := l.agentCreateDirectory(domName, filepath.Dir(filePath), "700", owner)
	if err != nil {
		return err
	}

	if fileExists {
		// ensure the file has the correct permissions for writing
		if permErr := l.agentSetFilePermissions(domName, filePath, "600", owner); permErr != nil {
			return permErr
		}
	}

	// write the file
	base64Str := base64.StdEncoding.EncodeToString([]byte(contents))
	cmdOpenFile := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": %q, "mode":"w" } }`, filePath)
	output, err := l.virConn.QemuAgentCommand(cmdOpenFile, domName)
	if err != nil {
		return err
	}

	openRes := &openReturn{}
	err = json.Unmarshal([]byte(output), openRes)
	if err != nil {
		return err
	}

	cmdWriteFile := fmt.Sprintf(`{"execute": "guest-file-write", "arguments": { "handle": %d, "buf-b64": %q } }`, openRes.Return, base64Str)
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
		if permErr := l.agentSetFilePermissions(domName, filePath, "600", owner); permErr != nil {
			return permErr
		}
	}

	return nil
}

func (l *AccessCredentialManager) readGuestFile(domName, filePath string) (string, error) {
	contents := ""

	cmdOpenFile := fmt.Sprintf(`{"execute": "guest-file-open", "arguments": { "path": %q, "mode":"r" } }`, filePath)
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
		readBytes, decodingErr := base64.StdEncoding.DecodeString(readRes.Return.BufB64)
		if decodingErr != nil {
			return contents, decodingErr
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

func (l *AccessCredentialManager) agentGuestExec(domName, command string, args []string) (string, error) {
	var timeoutInSeconds int32 = 10
	return agent.GuestExec(l.virConn, domName, command, args, timeoutInSeconds)
}

// Requires usage of mkdir, chown, chmod
func (l *AccessCredentialManager) agentCreateDirectory(domName, dir, permissions, owner string) error {
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

func (l *AccessCredentialManager) agentGetUserInfo(domName, user string) (filePath, uid, gid string, err error) {
	passwdEntryStr, err := l.agentGuestExec(domName, "getent", []string{"passwd", user})
	if err != nil {
		return "", "", "", fmt.Errorf("unable to detect home directory of user %s: %s", user, err.Error())
	}
	passwdEntryStr = strings.TrimSpace(passwdEntryStr)
	entries := strings.Split(passwdEntryStr, ":")

	const expectedPasswdEntries = 6
	if len(entries) < expectedPasswdEntries {
		return "", "", "", fmt.Errorf("unable to detect home directory of user %s", user)
	}

	filePath = entries[5]
	uid = entries[2]
	gid = entries[3]
	log.Log.Infof("Detected home directory %s for user %s", filePath, user)
	return filePath, uid, gid, nil
}

// Requires usage of chown, chmod
func (l *AccessCredentialManager) agentSetFilePermissions(domName, filePath, permissions, owner string) error {
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

func (l *AccessCredentialManager) agentSetUserPassword(domName, user, password string) (err error) {
	domain, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		return fmt.Errorf("domain lookup failed: %w", err)
	}
	defer func() { err = errors.Join(err, domain.Free()) }()

	return domain.SetUserPassword(user, password, 0)
}

func (l *AccessCredentialManager) pingAgent(domName string) error {
	cmdPing := `{"execute":"guest-ping"}`

	_, err := l.virConn.QemuAgentCommand(cmdPing, domName)
	return err
}

// agentSetAuthorizedKeys sets the SSH keys and returns "true" if the old deprecated flow was used.
func (l *AccessCredentialManager) agentSetAuthorizedKeys(domName, user string, authorizedKeys []string) (bool, error) {
	err := func() (err error) {
		domain, err := l.virConn.LookupDomainByName(domName)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, domain.Free()) }()

		// Zero flags argument means that the authorized_keys file is overwritten with the authorizedKeys
		return domain.AuthorizedSSHKeysSet(user, authorizedKeys, 0)
	}()
	if err == nil {
		return false, nil
	}

	log.Log.V(logVerbosityDebug).Infof("Could not set SSH key using guest-ssh-add-authorized-keys: %v", err)

	// If AuthorizedSSHKeysSet method failed, use the old method
	desiredAuthorizedKeys := strings.Join(authorizedKeys, "\n")
	secondErr := l.agentWriteAuthorizedKeysFile(domName, user, desiredAuthorizedKeys)
	if secondErr == nil {
		return true, nil
	}

	return false, fmt.Errorf(
		"failed to set SSH keys: error from guest-ssh-add-authorized-keys: %w; error from using guest-file-write: %w",
		err,
		secondErr,
	)
}

func (l *AccessCredentialManager) agentWriteAuthorizedKeysFile(domName, user, desiredAuthorizedKeys string) (err error) {
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

func (l *AccessCredentialManager) reportAccessCredentialResult(
	vmi *v1.VirtualMachineInstance,
	succeeded bool,
	message string,
	usedDeprecatedFlow bool,
) {
	acMetadata := api.AccessCredentialMetadata{
		Succeeded: succeeded,
		Message:   message,
	}
	l.metadataCache.AccessCredential.Store(acMetadata)
	log.Log.V(logVerbosityDebug).Infof("Access credential set in metadata: %v", acMetadata)

	if !succeeded || !usedDeprecatedFlow {
		l.eventSent = false
	}

	if succeeded && usedDeprecatedFlow && !l.eventSent {
		err := l.eventSender.SendK8sEvent(
			vmi,
			k8sv1.EventTypeWarning,
			v1.AccessCredentialsSyncSuccess.String(),
			"Used deprecated method to set SSH keys. It will be removed in a future release. Update qemu guest agent to 5.2 or newer to keep SSH key injection working.", //nolint:lll
		)
		if err != nil {
			log.Log.Reason(err).Errorf("Error encountered sending k8s event about using deperecated flow to update SSH key.")
		} else {
			l.eventSent = true
		}
	}
}

func (l *AccessCredentialManager) watchSecrets(vmi *v1.VirtualMachineInstance) {
	logger := log.Log.Object(vmi)

	reload := true
	fileChangeDetected := true

	domName := util.VMINamespaceKeyFunc(vmi)

	// guest agent will force a resync of changes every 'x' minutes
	const resyncInterval = 5 * time.Minute
	forceResyncTicker := time.NewTicker(resyncInterval)
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
			logger.Info("Signaled to stop watching access credential secrets")
			return
		}

		if !reload {
			continue
		}
		fileChangeDetected = false

		reload = l.reloadCredentialFiles(vmi, domName, logger)
	}
}

func (l *AccessCredentialManager) reloadCredentialFiles(vmi *v1.VirtualMachineInstance, domName string, logger *log.FilteredLogger) bool {
	err := l.pingAgent(domName)
	if err != nil {
		l.reportAccessCredentialResult(vmi, false, "Guest agent is offline", false)
		return true
	}

	reload := false
	reportedErr := false
	usedDeprecatedFlow := false

	credentialInfo := newAccessCredentialsInfo()

	// Step 1. Populate access credential info
	for i := range vmi.Spec.AccessCredentials {
		err := credentialInfo.addAccessCredential(&vmi.Spec.AccessCredentials[i])
		if err != nil {
			// if reading failed, reset reload to true so this change will be retried again
			reload = true
			reportedErr = true
			logger.Reason(err).Errorf("Error encountered")
			l.reportAccessCredentialResult(vmi, false, err.Error(), false)
		}
	}

	// Step 2. Update Authorized keys
	for user, secretNames := range credentialInfo.userSSHMap {
		var allAuthorizedKeys []string
		for _, secretName := range secretNames {
			pubKeys := credentialInfo.secretMap[secretName]
			allAuthorizedKeys = append(allAuthorizedKeys, pubKeys...)
		}

		deprecated, err := l.agentSetAuthorizedKeys(domName, user, allAuthorizedKeys)
		if err != nil {
			// if writing failed, reset reload to true so this change will be retried again
			reload = true
			reportedErr = true
			logger.Reason(err).Errorf("Error encountered writing access credentials using guest agent")
			l.reportAccessCredentialResult(vmi, false, fmt.Sprintf(
				"Error encountered writing ssh pub key access credentials for user [%s]: %v",
				user, err), false)
		}
		if deprecated {
			usedDeprecatedFlow = true
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

			l.reportAccessCredentialResult(vmi, false, fmt.Sprintf("Error encountered setting password for user [%s]: %v", user, err), false)
		}
	}
	if !reportedErr {
		l.reportAccessCredentialResult(vmi, true, "", usedDeprecatedFlow)
	}

	return reload
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
	if isSSHPublicKey(accessCred) {
		for _, user := range accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent.Users {
			a.userSSHMap[user] = append(a.userSSHMap[user], secretName)
		}

		authorizedKeys, err := readKeysFromDirectory(secretDir)
		if err != nil {
			return err
		}
		if len(authorizedKeys) > 0 {
			a.secretMap[secretName] = authorizedKeys
		}

		return nil
	}

	if isUserPassword(accessCred) {
		return readAndAddPasswordsFromDirectory(secretDir, a.userPasswordMap)
	}

	return nil
}

func readKeysFromDirectory(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error occurred while reading the list of secrets files from the base directory %s: %w", dir, err)
	}

	var authorizedKeys []string
	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
			continue
		}

		pubKeyBytes, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("error occurred while reading the access credential secret file [%s]: %w", filepath.Join(dir, file.Name()), err)
		}

		for _, pubKey := range strings.Split(string(pubKeyBytes), "\n") {
			trimmedKey := strings.TrimSpace(pubKey)
			if trimmedKey != "" {
				authorizedKeys = append(authorizedKeys, trimmedKey)
			}
		}
	}

	return authorizedKeys, nil
}

func readAndAddPasswordsFromDirectory(dir string, passMap map[string]string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error occurred while reading the list of secrets files from the base directory %s: %w", dir, err)
	}

	for _, file := range files {
		// Mounted secret directory contains directories prefixed by "..".
		// They are used by k8s to atomically swap all files when the secret is updated.
		if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
			continue
		}

		passwordBytes, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return fmt.Errorf("error occurred while reading the access credential secret file [%s]: %w", filepath.Join(dir, file.Name()), err)
		}

		password := strings.TrimSpace(string(passwordBytes))
		if password == "" {
			continue
		}
		passMap[file.Name()] = password
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
