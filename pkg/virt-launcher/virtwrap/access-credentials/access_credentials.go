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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

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

func (l *AccessCredentialManager) agentSetUserPassword(domName string, user string, password string) error {
	domain, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		return fmt.Errorf("domain lookup failed: %w", err)
	}
	defer domain.Free()
	return domain.SetUserPassword(user, password, 0)
}

func (l *AccessCredentialManager) pingAgent(domName string) error {
	cmdPing := `{"execute":"guest-ping"}`

	_, err := l.virConn.QemuAgentCommand(cmdPing, domName)
	return err
}

func (l *AccessCredentialManager) agentSetAuthorizedKeys(domName string, user string, authorizedKeys []string) (err error) {
	domain, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		return fmt.Errorf("failed to lookup domain %s: %w", domName, err)
	}
	defer func() {
		if freeErr := domain.Free(); err == nil && freeErr != nil {
			err = fmt.Errorf("failed to free domain %s: %w", domName, freeErr)
		}
	}()

	// Zero flags argument means that the authorized_keys file is overwritten with the authorizedKeys
	err = domain.AuthorizedSSHKeysSet(user, authorizedKeys, 0)
	if err != nil {
		return fmt.Errorf("failed to set SSH authorized keys: %w", err)
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
