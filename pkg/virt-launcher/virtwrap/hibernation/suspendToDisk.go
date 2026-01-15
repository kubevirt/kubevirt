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
package hibernation

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	FailedDomainSuspendToDisk = "Domain suspend to disk failed"
)

func (m *HibernationManager) SuspendToDisk(vmi *v1.VirtualMachineInstance) error {
	select {
	case m.HibernateInProgress <- struct{}{}:
	default:
		log.Log.Object(vmi).Infof("Suspend to disk is in progress")
		return nil
	}

	go func() {
		defer func() { <-m.HibernateInProgress }()
		if err := m.suspendToDisk(vmi); err != nil {
			log.Log.Object(vmi).Reason(err).Error(FailedDomainSuspendToDisk)
		}
	}()
	return nil
}

func (m *HibernationManager) suspendToDisk(vmi *v1.VirtualMachineInstance) error {
	logger := log.Log.Object(vmi)

	m.initializeHibernationMetadata()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if dom == nil || err != nil {
		return err
	}
	defer dom.Free()

	logger.Infof("Starting hibernate vm with suspendToDisk mode")
	failed := false
	reason := ""
	err = dom.PMSuspendForDuration(libvirt.NODE_SUSPEND_TARGET_DISK, 0, 0)
	if err != nil {
		failed = true
		reason = fmt.Sprintf("%s: %s", FailedDomainMemoryDump, err)
	} else {
		logger.Infof("Completed memory dump successfully")
	}

	m.setSuspendToDiskResult(failed, reason)
	return err
}

func (m *HibernationManager) initializeHibernationMetadata() {
	m.metadataCache.Hibernation.WithSafeBlock(func(hibernationMetadata *api.HibernationMetadata, initialized bool) {
		now := metav1.Now()
		*hibernationMetadata = api.HibernationMetadata{
			StartTimestamp: &now,
		}
	})
	log.Log.V(4).Infof("initialize hibernation metadata: %s", m.metadataCache.Hibernation.String())
}

func (m *HibernationManager) setSuspendToDiskResult(failed bool, reason string) {
	m.metadataCache.Hibernation.WithSafeBlock(func(hibernationMetadata *api.HibernationMetadata, initialized bool) {
		if !initialized {
			// nothing to report if hibernation metadata is empty
			return
		}

		now := metav1.Now()
		hibernationMetadata.EndTimestamp = &now
		hibernationMetadata.Failed = failed
		hibernationMetadata.FailureReason = reason
	})
	log.Log.V(4).Infof("set suspend to disk results in metadata: %s", m.metadataCache.Hibernation.String())
}
