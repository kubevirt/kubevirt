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

package storage

import (
	"fmt"
	"os/exec"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/tpm"
	"kubevirt.io/kubevirt/pkg/util"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func (m *StorageManager) FreezeVMI(vmi *v1.VirtualMachineInstance, unfreezeTimeoutSeconds int32) error {
	if m.MigrationInProgress() {
		return fmt.Errorf("failed to freeze VMI, VMI is currently during migration")
	}
	domainName := api.VMINamespaceKeyFunc(vmi)
	safetyUnfreezeTimeout := time.Duration(unfreezeTimeoutSeconds) * time.Second

	fsfreezeStatus, err := m.getParsedFSStatus(domainName)
	if err != nil {
		log.Log.Errorf("failed to get fs status before freeze vmi %s, %s", vmi.Name, err.Error())
		return err
	}

	isFrozen := fsfreezeStatus == api.FSFrozen
	hasTPM := tpm.HasPersistentDevice(&vmi.Spec)
	// idempotent - prevent failure in case fs is already frozen
	// allow exception if VM has TPM so we can retry syncing the swtpm mount in case previous attempt failed.
	if isFrozen && !hasTPM {
		return nil
	}

	if !isFrozen {
		domain, err := m.virConn.LookupDomainByName(domainName)
		if err != nil {
			log.Log.Errorf("Domain lookup failed: %v", err)
			return err
		}
		defer domain.Free()

		if err := domain.FSFreeze(nil, 0); err != nil {
			log.Log.Errorf("Failed to freeze vmi, %s", err.Error())
			return err
		}
	}

	m.cancelSafetyUnfreeze()
	if safetyUnfreezeTimeout != 0 {
		go m.scheduleSafetyVMIUnfreeze(vmi, safetyUnfreezeTimeout)
	}

	// Guest fsfreeze does not cover host-side swtpm state on the backend PVC.
	// use syncfs to flush the swtpm state to disk
	if hasTPM {
		// if sync fails, return error. subsequent calls to FreezeVMI will reattempt the sync.
		if err := syncFilesystem(util.PathForSwtpm(vmi)); err != nil {
			log.Log.Reason(err).Errorf("syncfs failed for swtpm directory")
			return err
		}
	}

	return nil
}

func (m *StorageManager) UnfreezeVMI(vmi *v1.VirtualMachineInstance) error {
	m.cancelSafetyUnfreeze()
	domainName := api.VMINamespaceKeyFunc(vmi)
	fsfreezeStatus, err := m.getParsedFSStatus(domainName)
	if err == nil {
		// prevent initating fs thaw to prevent rerunning the thaw hook
		if fsfreezeStatus == api.FSThawed {
			return nil
		}
	}

	domain, err := m.virConn.LookupDomainByName(domainName)
	if err != nil {
		log.Log.Errorf("Domain lookup failed: %v", err)
		return err
	}
	defer domain.Free()

	if err := domain.FSThaw(nil, 0); err != nil {
		log.Log.Errorf("Failed to unfreeze vmi, %s", err.Error())
		return err
	}
	return nil
}

func (m *StorageManager) scheduleSafetyVMIUnfreeze(vmi *v1.VirtualMachineInstance, unfreezeTimeout time.Duration) {
	select {
	case <-time.After(unfreezeTimeout):
		log.Log.Warningf("Unfreeze was not called for vmi %s for more then %v, initiating unfreeze",
			vmi.Name, unfreezeTimeout)
		m.UnfreezeVMI(vmi)
	case <-m.cancelSafetyUnfreezeChan:
		log.Log.V(3).Infof("Canceling scheduled Unfreeze for vmi %s", vmi.Name)
		// aborted
	}
}

func (m *StorageManager) cancelSafetyUnfreeze() {
	select {
	case m.cancelSafetyUnfreezeChan <- struct{}{}:
	default:
	}
}

func (m *StorageManager) getParsedFSStatus(domainName string) (string, error) {
	cmdResult, err := m.virConn.QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, domainName)
	if err != nil {
		return "", err
	}
	fsfreezeStatus, err := agentpoller.ParseFSFreezeStatus(cmdResult)
	if err != nil {
		return "", err
	}

	return fsfreezeStatus.Status, nil
}

// syncFilesystem runs syncfs on the filesystem containing path.
func syncFilesystem(path string) error {
	out, err := exec.Command("/usr/bin/sync", "-f", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("syncfs failed for %s: %w, output: %s", path, err, out)
	}
	return nil
}
