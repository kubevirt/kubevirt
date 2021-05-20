package dhcp

import (
	"fmt"
	"os"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

const dhcpStartedDirectory = "/var/run/kubevirt-private"

type Configurator struct {
	advertisingIfaceName string
	cacheFactory         cache.InterfaceCacheFactory
	filterByMac          bool
	handler              netdriver.NetworkHandler
	launcherPID          string
	dhcpStartedDirectory string
}

// NewConfiguratorWithClientFilter should be used when the DHCP server is
// expected to only reply to the MAC specified in the `cache.DhcpConfig` struct
func NewConfiguratorWithClientFilter(cacheFactory cache.InterfaceCacheFactory, launcherPID string, advertisingIfaceName string, handler netdriver.NetworkHandler) *Configurator {
	return &Configurator{
		advertisingIfaceName: advertisingIfaceName,
		cacheFactory:         cacheFactory,
		launcherPID:          launcherPID,
		filterByMac:          true,
		handler:              handler,
		dhcpStartedDirectory: dhcpStartedDirectory,
	}
}

// NewConfigurator should be used when the DHCP server is
// expected to reply all client requests, independently of their MAC address
func NewConfigurator(cacheFactory cache.InterfaceCacheFactory, launcherPID string, advertisingIfaceName string, handler netdriver.NetworkHandler) *Configurator {
	return &Configurator{
		advertisingIfaceName: advertisingIfaceName,
		cacheFactory:         cacheFactory,
		launcherPID:          launcherPID,
		filterByMac:          false,
		handler:              handler,
		dhcpStartedDirectory: dhcpStartedDirectory,
	}
}

func (d Configurator) ImportConfiguration(ifaceName string) (*cache.DhcpConfig, error) {
	dhcpConfig, err := d.cacheFactory.CacheDhcpConfigForPid(d.launcherPID).Read(ifaceName)
	if err != nil {
		return nil, err
	}
	dhcpConfig.AdvertisingIPAddr = dhcpConfig.AdvertisingIPAddr.To4()
	dhcpConfig.AdvertisingIPv6Addr = dhcpConfig.AdvertisingIPv6Addr.To16()
	return dhcpConfig, nil
}

func (d Configurator) ExportConfiguration(config cache.DhcpConfig) error {
	return d.cacheFactory.CacheDhcpConfigForPid(d.launcherPID).Write(config.Name, &config)
}

func (d Configurator) EnsureDhcpServerStarted(podInterfaceName string, dhcpConfig cache.DhcpConfig, dhcpOptions *v1.DHCPOptions) error {
	if dhcpConfig.IPAMDisabled {
		return nil
	}
	dhcpStartedFile := d.getDhcpStartedFilePath(podInterfaceName)
	_, err := os.Stat(dhcpStartedFile)
	if os.IsNotExist(err) {
		if err := d.handler.StartDHCP(&dhcpConfig, dhcpConfig.AdvertisingIPAddr, d.advertisingIfaceName, dhcpOptions, d.filterByMac); err != nil {
			return fmt.Errorf("failed to start DHCP server for interface %s", podInterfaceName)
		}
		newFile, err := os.Create(dhcpStartedFile)
		if err != nil {
			return fmt.Errorf("failed to create dhcp started file %s: %s", dhcpStartedFile, err)
		}
		newFile.Close()
	}
	return nil
}

func (d Configurator) getDhcpStartedFilePath(podInterfaceName string) string {
	return fmt.Sprintf("%s/dhcp_started-%s", d.dhcpStartedDirectory, podInterfaceName)
}
