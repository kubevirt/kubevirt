package network

import (
	"net"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-dhcp"
	"kubevirt.io/kubevirt/pkg/virt-handler/network/cniproxy"
	"kubevirt.io/kubevirt/pkg/virt-handler/network/utils"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

type PodNetworkInterface struct {
	cniProxy *cniproxy.CNIProxy
	Name     string
	IPAddr   net.IPNet
	Mac      net.HardwareAddr
	Gateway  net.IP
}

func (vif *PodNetworkInterface) Plug(domainManager virtwrap.DomainManager) error {
	var result types.Result

	netns, err := ns.GetNS(HostNetNS)
	if err != nil {
		log.Log.Reason(err).Error("failed to get host net namespace.")
		return err
	}

	iface, err := utils.GenerateRandomTapName()
	if err != nil {
		log.Log.Reason(err).Error("failed to generate random tap name")
		return err
	}

	// Create network interface
	err = netns.Do(func(_ ns.NetNS) error {
		vif.cniProxy, err = GetContainerInterface(iface)
		if err != nil {
			return err
		}
		res, err := vif.Create()
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		return err
	}

	r, err := types020.GetResult(result)

	if err != nil {
		return err
	}
	vif.IPAddr = r.IP4.IP
	vif.Gateway = r.IP4.Gateway.To4()

	// Switch to libvirt namespace
	libvNS, err := cniproxy.GetLibvirtNS()
	if err != nil {
		return err
	}
	libvnetns, err := ns.GetNS(libvNS.Net)
	if err != nil {
		log.Log.Reason(err).Error("failed to get libvirt net namespace")
		return err
	}
	libvnetns.Do(func(_ ns.NetNS) error {
		inter, err := utils.GetInterfaceByIP(vif.IPAddr.String())
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get interface by IP: %s", vif.IPAddr.String())
			return err
		}
		vif.Name = inter.Name
		link, err := netlink.LinkByName(inter.Name)
		if err != nil {
			log.Log.Reason(err).Errorf("error getting link for interface: %s", inter.Name)
			return err
		}
		err = netlink.AddrDel(link, &netlink.Addr{IPNet: &vif.IPAddr})

		if err != nil {
			log.Log.Reason(err).Errorf("failed to delete link for interface: %s", vif.Name)
			return err
		}

		// Set interface link to down to change its MAC address
		err = netlink.LinkSetDown(link)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", vif.Name)
			return err
		}

		vif.Mac, err = utils.ChangeMacAddr(vif.Name)
		if err != nil {
			return err
		}
		return nil
	})

	// Update DHCP
	dhcpClient, err := virtdhcp.GetClient()
	if err != nil {
		return err
	}
	dhcpClient.AddIP(vif.Mac.String(), vif.IPAddr.IP.String(), 86400)

	return nil
}

func (vif *PodNetworkInterface) Unplug(domainManager virtwrap.DomainManager) error {
	netns, err := ns.GetNS(HostNetNS)
	if err != nil {
		log.Log.Reason(err).Error("failed get host net namespace.")
		return err
	}

	// Remove network interface
	err = netns.Do(func(_ ns.NetNS) error {
		vif.cniProxy, err = GetContainerInterface(vif.Name)
		if err != nil {
			return err
		}
		err = vif.Remove()
		if err != nil {
			log.Log.Reason(err).Warningf("failed to delete container interface: %s", vif.Name)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Update DHCP
	dhcpClient, err := virtdhcp.GetClient()
	if err != nil {
		log.Log.Reason(err).Warning("failed to get dhcp client")
	}
	dhcpClient.RemoveIP(vif.Mac.String(), vif.IPAddr.IP.String())
	return nil
}

func (vif *PodNetworkInterface) Remove() error {
	err := vif.cniProxy.DeleteFromNetwork()
	if err != nil {
		return err
	}
	return nil
}

func (vif *PodNetworkInterface) Create() (types.Result, error) {
	res, err := vif.cniProxy.AddToNetwork()
	if err != nil {
		log.Log.Reason(err).Error("failed to create container interface")
		return nil, err
	}
	return res, nil
}

func (vif *PodNetworkInterface) DecorateInterfaceMetadata() *v1.MetadataDevice {
	inter := v1.MetadataDevice{
		Type:   "PodNetworking",
		Device: vif.Name,
		Mac:    vif.Mac.String(),
		IP:     vif.IPAddr.IP.String(),
	}
	return &inter
}

func (vif *PodNetworkInterface) GetConfig() (*v1.Interface, error) {

	inter := v1.Interface{}
	inter.Type = "direct"
	inter.TrustGuestRxFilters = "yes"
	inter.Source = v1.InterfaceSource{Device: vif.Name, Mode: "passthrough"}
	inter.MAC = &v1.MAC{MAC: vif.Mac.String()}
	inter.Model = &v1.Model{Type: "virtio"}

	return &inter, nil
}

func (vif *PodNetworkInterface) SetInterfaceAttributes(mac string, ip string, device string) error {
	setMac, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}

	vif.Mac = setMac
	vif.IPAddr.IP = net.ParseIP(ip)
	vif.Name = device

	return nil
}
