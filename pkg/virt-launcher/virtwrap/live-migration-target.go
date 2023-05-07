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
	"path/filepath"
	"strconv"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func (l *LibvirtDomainManager) finalizeMigrationTarget(vmi *v1.VirtualMachineInstance) error {
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

func canSourceMigrateOverUnixURI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.MigrationTransport == v1.MigrationTransportUnix
}

func (l *LibvirtDomainManager) prepareMigrationTarget(
	vmi *v1.VirtualMachineInstance,
	allowEmulation bool,
	options *cmdv1.VirtualMachineOptions,
) error {
	logger := log.Log.Object(vmi)

	c, err := l.generateConverterContext(vmi, allowEmulation, options, true)
	if err != nil {
		return fmt.Errorf("Failed to generate libvirt domain from VMI spec: %v", err)
	}

	domain := &api.Domain{}
	if err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c); err != nil {
		return fmt.Errorf("conversion failed: %v", err)
	}

	dom, err := l.preStartHook(vmi, domain, true)
	if err != nil {
		return fmt.Errorf("pre-start pod-setup failed: %v", err)
	}

	l.metadataCache.UID.Set(vmi.UID)
	l.metadataCache.GracePeriod.Set(
		api.GracePeriodMetadata{DeletionGracePeriodSeconds: converter.GracePeriodSeconds(vmi)},
	)

	err = l.generateCloudInitEmptyISO(vmi, nil)
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

	if shouldBlockMigrationTargetPreparation(vmi) {
		return fmt.Errorf("Blocking preparation of migration target in order to satisfy a functional test condition")
	}

	if canSourceMigrateOverUnixURI(vmi) {
		// Prepare the directory for migration sockets
		migrationSocketsPath := filepath.Join(l.virtShareDir, "migrationproxy")
		err = util.MkdirAllWithNosec(migrationSocketsPath)
		if err != nil {
			logger.Reason(err).Error("failed to create the migration sockets directory")
			return err
		}
		if err := diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(migrationSocketsPath); err != nil {
			logger.Reason(err).Error("failed to change ownership on migration sockets directory")
			return err
		}
	} else {
		logger.V(3).Info("Setting up TCP proxies to support incoming legacy VMI migration")
		loopbackAddress := ip.GetLoopbackAddress()

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
	}

	// since the source vmi is paused, add the vmi uuid to the pausedVMIs as
	// after the migration this vmi should remain paused.
	if vmiHasCondition(vmi, v1.VirtualMachineInstancePaused) {
		logger.V(3).Info("adding vmi uuid to pausedVMIs list on the target")
		l.paused.add(vmi.UID)
	}

	return nil
}
