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

	// idempotent - prevent failure in case fs is already frozen
	if fsfreezeStatus == api.FSFrozen {
		return nil
	}

	// The fsfreeze doesn't apply to the TPM, so we can at least do a fsync to the state
	// directory to ensure data integrity. This explicit sync ensures that pending
	// writes to the swtpm backing files are flushed to disk.
	if tpm.HasPersistentDevice(&vmi.Spec) {
		cmd := exec.Command("/usr/bin/sync", util.PathForSwtpm(vmi))
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Log.Errorf("fsync error to TPM state directory: %s, output: %s", err.Error(), out)
			return err
		}
	}

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

	m.cancelSafetyUnfreeze()
	if safetyUnfreezeTimeout != 0 {
		go m.scheduleSafetyVMIUnfreeze(vmi, safetyUnfreezeTimeout)
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
		log.Log.V(3).Infof("Canceling schedualed Unfreeze for vmi %s", vmi.Name)
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
