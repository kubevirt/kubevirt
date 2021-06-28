package dhcp

import (
	"kubevirt.io/kubevirt/pkg/network/cache"
)

type BridgeConfigGenerator struct {
	podInterfaceName string
	cacheFactory     cache.InterfaceCacheFactory
	launcherPID      string
}

func (d *BridgeConfigGenerator) Generate() (*cache.DHCPConfig, error) {
	return d.cacheFactory.CacheDHCPConfigForPid(d.launcherPID).Read(d.podInterfaceName)
}
