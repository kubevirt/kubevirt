package dhcp

import (
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

type BridgeConfigGenerator struct {
	handler          netdriver.NetworkHandler
	podInterfaceName string
	cacheFactory     cache.InterfaceCacheFactory
	launcherPID      string
	vmiSpecIfaces    []v1.Interface
	vmiSpecIface     *v1.Interface
	subdomain        string
}

func (d *BridgeConfigGenerator) Generate() (*cache.DHCPConfig, error) {
	dhcpConfig, err := d.cacheFactory.CacheDHCPConfigForPid(d.launcherPID).Read(d.podInterfaceName)
	if err != nil {
		return nil, err
	}

	if dhcpConfig.IPAMDisabled {
		return dhcpConfig, nil
	}

	dhcpConfig.Name = d.podInterfaceName

	fakeBridgeIP := virtnetlink.GetFakeBridgeIP(d.vmiSpecIfaces, d.vmiSpecIface)
	fakeServerAddr, _ := netlink.ParseAddr(fakeBridgeIP)
	dhcpConfig.AdvertisingIPAddr = fakeServerAddr.IP

	newPodNicName := virtnetlink.GenerateNewBridgedVmiInterfaceName(d.podInterfaceName)
	podNicLink, err := d.handler.LinkByName(newPodNicName)
	if err != nil {
		return nil, err
	}
	dhcpConfig.Mtu = uint16(podNicLink.Attrs().MTU)
	dhcpConfig.Subdomain = d.subdomain

	return dhcpConfig, nil
}
