package dhcp

import (
	"fmt"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

type Configurator struct {
	advertisingIfaceName string
	cacheFactory         cache.InterfaceCacheFactory
	filterByMac          bool
	handler              netdriver.NetworkHandler
	launcherPID          string
}

// NewConfiguratorWithClientFilter should be used when the DHCP server is
// expected to only reply to the MAC specified in the `cache.DHCPConfig` struct
func NewConfiguratorWithClientFilter(cacheFactory cache.InterfaceCacheFactory, launcherPID string, advertisingIfaceName string, handler netdriver.NetworkHandler) *Configurator {
	return &Configurator{
		advertisingIfaceName: advertisingIfaceName,
		cacheFactory:         cacheFactory,
		launcherPID:          launcherPID,
		filterByMac:          true,
		handler:              handler,
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
	}
}

func (d Configurator) ImportConfiguration(ifaceName string) (*cache.DHCPConfig, error) {
	dhcpConfig, err := d.cacheFactory.CacheDHCPConfigForPid(d.launcherPID).Read(ifaceName)
	if err != nil {
		return nil, err
	}
	dhcpConfig.AdvertisingIPAddr = dhcpConfig.AdvertisingIPAddr.To4()
	dhcpConfig.Gateway = dhcpConfig.Gateway.To4()
	dhcpConfig.AdvertisingIPv6Addr = dhcpConfig.AdvertisingIPv6Addr.To16()
	return dhcpConfig, nil
}

func (d Configurator) ExportConfiguration(config cache.DHCPConfig) error {
	return d.cacheFactory.CacheDHCPConfigForPid(d.launcherPID).Write(config.Name, &config)
}

func (d Configurator) StartDHCPServer(podInterfaceName string, dhcpConfig cache.DHCPConfig, dhcpOptions *v1.DHCPOptions) error {
	if dhcpConfig.IPAMDisabled {
		return nil
	}
	if err := d.handler.StartDHCP(&dhcpConfig, d.advertisingIfaceName, dhcpOptions, d.filterByMac); err != nil {
		return fmt.Errorf("failed to start DHCP server for interface %s", podInterfaceName)
	}
	return nil
}
