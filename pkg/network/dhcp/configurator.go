package dhcp

import (
	"kubevirt.io/kubevirt/pkg/network/cache"
)

type Configurator struct {
	cacheFactory cache.InterfaceCacheFactory
	launcherPID  string
}

func NewConfigurator(configurationFactory cache.InterfaceCacheFactory, launcherPID string) Configurator {
	return Configurator{
		cacheFactory: configurationFactory,
		launcherPID:  launcherPID,
	}
}

func (d Configurator) ImportConfiguration(ifaceName string) (*cache.DhcpConfig, error) {
	dhcpConfig, err := d.cacheFactory.CacheDhcpConfigForPid(d.launcherPID).Read(ifaceName)
	if err != nil {
		return nil, err
	}
	dhcpConfig.Gateway = dhcpConfig.Gateway.To4()
	dhcpConfig.GatewayIpv6 = dhcpConfig.GatewayIpv6.To16()
	return dhcpConfig, nil
}

func (d Configurator) ExportConfiguration(config cache.DhcpConfig) error {
	return d.cacheFactory.CacheDhcpConfigForPid(d.launcherPID).Write(config.Name, &config)
}
