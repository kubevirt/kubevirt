// pkg/virt-launcher/virtwrap/network/cloud-init-network.go currently adds
// support to configure SR-IOV interfaces within a VM through cloud-init
// network version 1 configuration. Other interface types such as bridge
// are configured within the VM by binding a DHCP server to the bridge
// source interface in the compute container. This is not possible for
// SR-IOV network interfaces as there is nothing in the compute container
// to bind a DHCP server to.

// Other network interface types can be added to this logic but are
// currently already handled with existing code. This code does not
// currently interfere with existing functionality.

package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
	"gopkg.in/yaml.v2"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type CloudInitNetworkInterface struct {
	NetworkType string            `yaml:"type"`
	Name        string            `yaml:"name,omitempty"`
	MacAddress  string            `yaml:"mac_address,omitempty"`
	Mtu         uint16            `yaml:"mtu,omitempty"`
	Subnets     []CloudInitSubnet `yaml:"subnets,omitempty"`
	Address     []string          `yaml:"address,omitempty"`
	Search      []string          `yaml:"search,omitempty"`
	Destination string            `yaml:"destination,omitempty"`
	Gateway     string            `yaml:"gateway,omitempty"`
	Metric      int               `yaml:"metric,omitempty"`
}

type CloudInitSubnet struct {
	SubnetType string                 `yaml:"type"`
	Address    string                 `yaml:"address"`
	Gateway    string                 `yaml:"gateway,omitempty"`
	Routes     []CloudInitSubnetRoute `yaml:"routes,omitempty"`
}

type CloudInitSubnetRoute struct {
	Network string `yaml:"network,omitempty"`
	Netmask string `yaml:"netmask,omitempty"`
	Gateway string `yaml:"gateway,omitempty"`
}

type CloudInitNetConfig struct {
	Version int                         `yaml:"version"`
	Config  []CloudInitNetworkInterface `yaml:"config"`
}

type CloudInitManageResolv struct {
	ManageResolv bool                `yaml:"manage_resolv_conf"`
	ResolvConf   CloudInitResolvConf `yaml:"resolv_conf"`
}

type CloudInitResolvConf struct {
	NameServers   []string `yaml:"nameservers,omitempty"`
	SearchDomains []string `yaml:"searchdomains,omitempty"`
	Domain        string   `yaml:"domain,omitempty"`
	// TODO Add options map when pkg/util/net/dns can parse them
}

const cloudInitDelimiter = "###CLOUDINITDELIMITER###"

// Borrowed from Convert_v1_VirtualMachine_To_api_Domain
func getSriovNetworkInfo(vmi *v1.VirtualMachineInstance) ([]VIF, error) {
	networks := map[string]*v1.Network{}
	cniNetworks := map[string]int{}
	var sriovVifs []VIF

	for _, network := range vmi.Spec.Networks {
		if network.Multus != nil {
			cniNetworks[network.Name] = len(cniNetworks) + 1
		}
		if network.Genie != nil {
			cniNetworks[network.Name] = len(cniNetworks)
		}
		networks[network.Name] = network.DeepCopy()
	}

	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		net, isExist := networks[iface.Name]
		if !isExist {
			return sriovVifs, fmt.Errorf("failed to find network %s", iface.Name)
		}

		if value, ok := cniNetworks[iface.Name]; ok {
			prefix := ""
			// no error check, we assume that CNI type was set correctly
			if net.Multus != nil {
				prefix = "net"
			} else if net.Genie != nil {
				prefix = "eth"
			}
			if iface.SRIOV != nil {
				details, err := getNetworkDetails(fmt.Sprintf("%s%d", prefix, value))
				if err != nil {
					log.Log.Reason(err).Errorf("failed to get SR-IOV network details for %s", fmt.Sprintf("%s%d", prefix, value))
					return sriovVifs, err
				}
				sriovVifs = append(sriovVifs, details)
			}
		}
	}

	return sriovVifs, nil
}

// Scavenged from various parts of podnetwork and BridgePodInterface
func getNetworkDetails(intName string) (VIF, error) {
	initHandler()
	var vif VIF
	vif.Name = intName
	link, err := Handler.LinkByName(vif.Name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", vif.Name)
		return vif, err
	}

	// get IP address
	addrList, err := Handler.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", vif.Name)
		return vif, err
	}
	// TODO: This can return an empty object. Test for it and skip assignment.
	// This results in subnet: being set to <nil>
	if len(addrList) > 0 {
		vif.IP = addrList[0]
	}

	if len(vif.MAC) == 0 {
		// Get interface MAC address
		mac, err := Handler.GetMacDetails(vif.Name)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get MAC for %s", vif.Name)
			return vif, err
		}
		vif.MAC = mac
	}

	// Get interface MTU
	vif.Mtu = uint16(link.Attrs().MTU)
	routes, err := Handler.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", vif.Name)
		return vif, err
	}
	vif.Routes = &routes

	return vif, nil
}

func getCloudInitManageResolv() CloudInitManageResolv {
	var cloudInitManageResolv CloudInitManageResolv
	var cloudInitResolvConf CloudInitResolvConf

	nameServers, searchDomains, err := api.GetResolvConfDetailsFromPod()
	if err != nil {
		log.Log.Errorf("Failed to get DNS servers from resolv.conf: %v", err)
		panic(err)
	}

	cloudInitManageResolv.ManageResolv = true

	for _, nameServer := range nameServers {
		cloudInitResolvConf.NameServers = append(cloudInitResolvConf.NameServers, net.IP(nameServer).String())
	}

	for _, searchDomain := range searchDomains {
		cloudInitResolvConf.SearchDomains = append(cloudInitResolvConf.SearchDomains, searchDomain)
	}

	cloudInitManageResolv.ResolvConf = cloudInitResolvConf

	return cloudInitManageResolv
}

func CloudInitDiscoverNetworkData(vmi *v1.VirtualMachineInstance) ([]byte, error) {
	var networkFile []byte
	var resolvFile []byte
	var cloudInitNetworks []VIF

	sriovNetworks, err := getSriovNetworkInfo(vmi)
	if err != nil {
		return networkFile, err
	}

	if len(sriovNetworks) > 0 {
		cloudInitNetworks = append(cloudInitNetworks, sriovNetworks...)
	}

	// More options for getting network info could be added here

	if len(cloudInitNetworks) == 0 {
		return networkFile, err
	}

	var config = CloudInitNetConfig{
		Version: 1,
	}

	for _, vif := range cloudInitNetworks {
		var nif CloudInitNetworkInterface
		var nifSubnet CloudInitSubnet
		var nifRoutes []CloudInitSubnetRoute

		nif.Name = vif.Name
		nif.NetworkType = "physical"
		nif.MacAddress = vif.MAC.String()
		nif.Mtu = vif.Mtu

		nifSubnet.SubnetType = "static"
		nifSubnet.Address = strings.Split(vif.IP.String(), " ")[0]
		if vif.Gateway != nil {
			nifSubnet.Gateway = string(vif.Gateway)
		}
		for _, route := range *vif.Routes {
			if route.Dst == nil && route.Src.Equal(nil) && route.Gw.Equal(nil) {
				continue
			}

			if route.Src != nil && route.Src.Equal(vif.IP.IP) {
				continue
			}

			var subnetRoute CloudInitSubnetRoute

			if route.Dst == nil {
				nifSubnet.Gateway = route.Gw.String()
				continue
			} else {
				subnetRoute.Network = route.Dst.IP.String()
			}

			subnetRoute.Network = route.Dst.IP.String()
			subnetRoute.Netmask = net.IP(route.Dst.Mask).String()
			if route.Gw != nil {
				subnetRoute.Gateway = route.Gw.String()
			}
			nifRoutes = append(nifRoutes, subnetRoute)
		}
		nifSubnet.Routes = nifRoutes
		nif.Subnets = append(nif.Subnets, nifSubnet)
		config.Config = append(config.Config, nif)
	}

	networkFile, err = yaml.Marshal(config)
	if err != nil {
		return networkFile, err
	}

	// Get resolver configuration. dhclient will likely override this on most
	// distributions but it is the same data so this should be safe.
	// This can be gated via Spec if needed.
	cloudInitManageResolv := getCloudInitManageResolv()
	resolvFile, err = yaml.Marshal(cloudInitManageResolv)
	if err != nil {
		return networkFile, err
	}

	// Append resolv conf to network file with a delimiter so we can split
	// it later.
	if len(resolvFile) > 0 {
		networkFile = append(networkFile, []byte(cloudInitDelimiter)...)
		networkFile = append(networkFile, resolvFile...)
	}

	return networkFile, err
}
