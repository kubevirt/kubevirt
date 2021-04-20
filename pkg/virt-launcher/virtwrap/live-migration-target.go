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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package virtwrap

import (
	"fmt"
	"net"
	"strconv"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func (l *LibvirtDomainManager) finalizeMigrationTarget(vmi *v1.VirtualMachineInstance) error {
	if err := l.hotPlugHostDevices(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to hot-plug host-devices")
	}

	if err := l.setGuestTime(vmi); err != nil {
		return err
	}

	return nil
}

func shouldBlockMigrationTargetPreparation(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Annotations == nil {
		return false
	}

	_, shouldBlock := vmi.Annotations[v1.FuncTestBlockLauncherPrepareMigrationTargetAnnotation]
	return shouldBlock
}

func (l *LibvirtDomainManager) prepareMigrationTarget(vmi *v1.VirtualMachineInstance, useEmulation bool) error {
	logger := log.Log.Object(vmi)

	if shouldBlockMigrationTargetPreparation(vmi) {
		return fmt.Errorf("Blocking preparation of migration target in order to satisfy a functional test condition")
	}

	c, err := l.generateConverterContext(vmi, useEmulation, nil, true)
	if err != nil {
		return fmt.Errorf("Failed to generate libvirt domain from VMI spec: %v", err)
	}

	domain := &api.Domain{}
	if err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c); err != nil {
		return fmt.Errorf("conversion failed: %v", err)
	}

	dom, err := l.preStartHook(vmi, domain)
	if err != nil {
		return fmt.Errorf("pre-start pod-setup failed: %v", err)
	}

	err = l.generateCloudInitISO(vmi, nil)
	if err != nil {
		return err
	}
	// TODO this should probably a OnPrepareMigration hook or something.
	// Right now we need to call OnDefineDomain, so that additional setup, which might be done
	// by the hook can also be done for the new target pod
	hooksManager := hooks.GetManager()
	_, err = hooksManager.OnDefineDomain(&dom.Spec, vmi)
	if err != nil {
		return fmt.Errorf("executing custom preStart hooks failed: %v", err)
	}

	loopbackAddress := ip.GetLoopbackAddress()
	if err := updateHostsFile(fmt.Sprintf("%s %s\n", loopbackAddress, vmi.Status.MigrationState.TargetPod)); err != nil {
		return fmt.Errorf("failed to update the hosts file: %v", err)
	}

	migrationPortsRange := migrationproxy.GetMigrationPortsList(isBlockMigration(vmi))
	for _, port := range migrationPortsRange {
		// Prepare the direct migration proxy
		key := migrationproxy.ConstructProxyKey(string(vmi.UID), port)
		curDirectAddress := net.JoinHostPort(loopbackAddress, strconv.Itoa(port))
		unixSocketPath := migrationproxy.SourceUnixFile(l.virtShareDir, key)
		migrationProxy := migrationproxy.NewSourceProxy(unixSocketPath, curDirectAddress, nil, nil, string(vmi.UID))

		err := migrationProxy.Start()
		if err != nil {
			logger.Reason(err).Errorf("proxy listening failed, socket %s", unixSocketPath)
			return err
		}
	}

	// since the source vmi is paused, add the vmi uuid to the pausedVMIs as
	// after the migration this vmi should remain paused.
	if vmiHasCondition(vmi, v1.VirtualMachineInstancePaused) {
		log.Log.Object(vmi).V(3).Info("adding vmi uuid to pausedVMIs list on the target")
		l.paused.add(vmi.UID)
	}

	return nil
}
