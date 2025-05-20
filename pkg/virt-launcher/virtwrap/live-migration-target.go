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
	"encoding/xml"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func (l *LibvirtDomainManager) finalizeMigrationTarget(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error {
	interfacesToReconnect := interfacesToReconnect(options)
	if len(interfacesToReconnect) != 0 {
		if err := l.reconnectGuestNics(vmi, interfacesToReconnect); err != nil {
			return err
		}
	}

	if err := l.setGuestTime(vmi); err != nil {
		return err
	}

	return nil
}

func interfacesToReconnect(options *cmdv1.VirtualMachineOptions) map[string]struct{} {
	interfaceMigrationOptions := options.InterfaceMigration
	ifacesToRefresh := map[string]struct{}{}
	for ifaceName, migrationOption := range interfaceMigrationOptions {
		if migrationOption != nil && migrationOption.Method == string(v1.LinkRefresh) {
			ifacesToRefresh[ifaceName] = struct{}{}
		}
	}
	return ifacesToRefresh
}

// reconnectGuestNics sets interfaces link down and up to renew DHCP leases
func (l *LibvirtDomainManager) reconnectGuestNics(vmi *v1.VirtualMachineInstance, ifacesToRefresh map[string]struct{}) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to reconnect guest interfaces")
		return err
	}
	defer dom.Free()
	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return err
	}

	var domain api.DomainSpec
	if err = xml.Unmarshal([]byte(xmlstr), &domain); err != nil {
		return fmt.Errorf("parsing domain XML failed, err: %v", err)
	}

	// Look up all the interfaces and reconnect them
	for _, iface := range domain.Devices.Interfaces {
		if _, exist := ifacesToRefresh[iface.Alias.GetName()]; !exist {
			continue
		}

		if err = reconnectIface(dom, iface); err != nil {
			return fmt.Errorf("failed to update network %s, err: %v", iface.Alias.GetName(), err)
		}
	}

	return nil
}

// reconnectIface sets link down and up for specified interface
func reconnectIface(dom cli.VirDomain, iface api.Interface) error {
	ifaceBytes, err := xml.Marshal(iface)
	if err != nil {
		return fmt.Errorf("failed to encode (xml) interface, err: %v", err)
	}
	disconnectedIface := iface.DeepCopy()
	disconnectedIface.LinkState = &api.LinkState{State: "down"}
	disconnectedIfaceBytes, err := xml.Marshal(disconnectedIface)
	if err != nil {
		return fmt.Errorf("failed to encode (xml) interface, err: %v", err)
	}
	if err = dom.UpdateDeviceFlags(string(disconnectedIfaceBytes), affectDeviceLiveAndConfigLibvirtFlags); err != nil {
		return fmt.Errorf("failed to set link down, err: %v", err)
	}

	// the sleep is needed since setting the interface immediately back to up may cause some guest OSs to ignore the both down and up requests
	time.Sleep(100 * time.Millisecond)

	if err = dom.UpdateDeviceFlags(string(ifaceBytes), affectDeviceLiveAndConfigLibvirtFlags); err != nil {
		return fmt.Errorf("failed to set link up, err: %v", err)
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

	dom, err := l.preStartHook(vmi, domain, true, options)
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
