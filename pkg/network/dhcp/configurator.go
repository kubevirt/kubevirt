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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package dhcp

import (
	"errors"
	"fmt"
	"os"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

const defaultDHCPStartedDirectory = "/var/run/kubevirt-private"

type Configurator interface {
	EnsureDHCPServerStarted(podInterfaceName string, dhcpConfig cache.DHCPConfig, dhcpOptions *v1.DHCPOptions) error
	Generate() (*cache.DHCPConfig, error)
}

type configurator struct {
	advertisingIfaceName string
	configGenerator      ConfigGenerator
	handler              netdriver.NetworkHandler
	dhcpStartedDirectory string
	podInterfaceName     string
}

type ConfigGenerator interface {
	Generate() (*cache.DHCPConfig, error)
}

func NewBridgeConfigurator(cacheCreator cacheCreator, launcherPID string, advertisingIfaceName string, handler netdriver.NetworkHandler, podInterfaceName string,
	vmiSpecIfaces []v1.Interface, vmiSpecIface *v1.Interface, subdomain string,
) *configurator {
	return &configurator{
		podInterfaceName:     podInterfaceName,
		advertisingIfaceName: advertisingIfaceName,
		handler:              handler,
		dhcpStartedDirectory: defaultDHCPStartedDirectory,
		configGenerator: &BridgeConfigGenerator{
			handler:          handler,
			podInterfaceName: podInterfaceName,
			cacheCreator:     cacheCreator,
			launcherPID:      launcherPID,
			vmiSpecIfaces:    vmiSpecIfaces,
			vmiSpecIface:     vmiSpecIface,
			subdomain:        subdomain,
		},
	}
}

func NewMasqueradeConfigurator(advertisingIfaceName string, handler netdriver.NetworkHandler, vmiSpecIface *v1.Interface, vmiSpecNetwork *v1.Network, podInterfaceName string,
	subdomain string,
) *configurator {
	return &configurator{
		podInterfaceName:     podInterfaceName,
		advertisingIfaceName: advertisingIfaceName,
		configGenerator: &MasqueradeConfigGenerator{
			handler: handler, vmiSpecIface: vmiSpecIface, vmiSpecNetwork: vmiSpecNetwork,
			subdomain: subdomain, podInterfaceName: podInterfaceName,
		},
		handler:              handler,
		dhcpStartedDirectory: defaultDHCPStartedDirectory,
	}
}

func (d *configurator) EnsureDHCPServerStarted(podInterfaceName string, dhcpConfig cache.DHCPConfig, dhcpOptions *v1.DHCPOptions) error {
	if dhcpConfig.IPAMDisabled {
		return nil
	}
	dhcpStartedFile := d.getDHCPStartedFilePath(podInterfaceName)
	_, err := os.Stat(dhcpStartedFile)
	if errors.Is(err, os.ErrNotExist) {
		if err := d.handler.StartDHCP(&dhcpConfig, d.advertisingIfaceName, dhcpOptions); err != nil {
			return fmt.Errorf("failed to start DHCP server for interface %s", podInterfaceName)
		}
		newFile, err := os.Create(dhcpStartedFile)
		if err != nil {
			return fmt.Errorf("failed to create dhcp started file %s: %s", dhcpStartedFile, err)
		}

		if err := newFile.Close(); err != nil {
			log.Log.Warningf(
				"failed to close the DHCP readiness file descriptor %d: %v", int(newFile.Fd()), err)
		}
	}
	return nil
}

func (d *configurator) getDHCPStartedFilePath(podInterfaceName string) string {
	return fmt.Sprintf("%s/dhcp_started-%s", d.dhcpStartedDirectory, podInterfaceName)
}

func (d *configurator) Generate() (*cache.DHCPConfig, error) {
	return d.configGenerator.Generate()
}
