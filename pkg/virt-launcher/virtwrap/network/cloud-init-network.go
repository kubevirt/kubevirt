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
	Mac_address string            `yaml:"mac_address,omitempty"`
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

type CloudInitConfig struct {
	Version int                         `yaml:"version"`
	Config  []CloudInitNetworkInterface `yaml:"config"`
}

// Borrowed from Convert_v1_VirtualMachine_To_api_Domain
func getSriovNetworkInfo(vmi *v1.VirtualMachineInstance) ([]VIF, error) {
	networks := map[string]*v1.Network{}
	cniNetworks := map[string]int{}
	multusNetworkIndex := 1
	var sriovVifs []VIF

	for _, network := range vmi.Spec.Networks {
		numberOfSources := 0
		if network.Pod != nil {
			numberOfSources++
		}
		if network.Multus != nil {
			if network.Multus.Default {
				// default network is eth0
				cniNetworks[network.Name] = 0
			} else {
				cniNetworks[network.Name] = multusNetworkIndex
				multusNetworkIndex++
			}
			numberOfSources++
		}
		if network.Genie != nil {
			cniNetworks[network.Name] = len(cniNetworks)
			numberOfSources++
		}
		if numberOfSources == 0 {
			return sriovVifs, fmt.Errorf("fail network %s must have a network type", network.Name)
		} else if numberOfSources > 1 {
			return sriovVifs, fmt.Errorf("fail network %s must have only one network type", network.Name)
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
				if net.Multus.Default {
					// Default network is eth0
					prefix = "eth"
				} else {
					prefix = "net"
				}
			} else if net.Genie != nil {
				prefix = "eth"
			}
			if iface.SRIOV != nil {
				details, err := discoverSriovNetworkInterface(fmt.Sprintf("%s%d", prefix, value))
				if err != nil {
					log.Log.Reason(err).Errorf("failed to get sriov network details for %s", fmt.Sprintf("%s%d", prefix, value))
					return sriovVifs, err
				}
				sriovVifs = append(sriovVifs, details)
			}
		}
	}

	return sriovVifs, nil
}

// Scavenged from various parts of podnetwork and BridgePodInterface
// TODO see if some of the calls during Plug should be abstracted out.
func discoverSriovNetworkInterface(intName string) (VIF, error) {
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

func setCloudInitResolv() CloudInitNetworkInterface {
	var cloudInitResolv CloudInitNetworkInterface

	nameServers, searchDomains, err := api.GetResolvConfDetailsFromPod()
	if err != nil {
		log.Log.Errorf("Failed to get DNS servers from resolv.conf: %v", err)
		panic(err)
	}

	cloudInitResolv.NetworkType = "nameserver"

	for _, nameServer := range nameServers {
		cloudInitResolv.Address = append(cloudInitResolv.Address, net.IP(nameServer).String())
	}

	for _, searchDomain := range searchDomains {
		cloudInitResolv.Search = append(cloudInitResolv.Search, searchDomain)
	}

	return cloudInitResolv
}

func GenNetworkFile(vmi *v1.VirtualMachineInstance) ([]byte, error) {
	var networkFile []byte
	var cloudInitNetworks []VIF

	sriovNetworks, err := getSriovNetworkInfo(vmi)
	if err != nil {
		return networkFile, err
	}

	if len(sriovNetworks) > 0 {
		cloudInitNetworks = append(cloudInitNetworks, sriovNetworks...)
	}

	// More options for getting network info could be added here
	// E.G. Static configurations from vmi SPEC

	if len(cloudInitNetworks) == 0 {
		return networkFile, err
	}

	var config = CloudInitConfig{
		Version: 1,
	}

	for _, vif := range cloudInitNetworks {
		var nif CloudInitNetworkInterface
		var nifSubnet CloudInitSubnet
		var nifRoutes []CloudInitSubnetRoute

		nif.Name = vif.Name
		nif.NetworkType = "physical"
		nif.Mac_address = vif.MAC.String()
		nif.Mtu = vif.Mtu

		nifSubnet.SubnetType = "static"
		nifSubnet.Address = strings.Split(vif.IP.String(), " ")[0]
		if vif.Gateway != nil {
			nifSubnet.Gateway = string(vif.Gateway)
		}
		for _, route := range *vif.Routes {
			if route.Gw == nil {
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
			subnetRoute.Gateway = route.Gw.String()
			nifRoutes = append(nifRoutes, subnetRoute)
		}
		nifSubnet.Routes = nifRoutes
		nif.Subnets = append(nif.Subnets, nifSubnet)
		config.Config = append(config.Config, nif)
	}

	// Get resolver configuration. dhclient will likely override this on most
	// distrobutions but it is the same data so this should be safe.
	// This can be gated via Spec if needed.
	cloudInitResolv := setCloudInitResolv()

	config.Config = append(config.Config, cloudInitResolv)

	networkFile, err = yaml.Marshal(config)

	if err != nil {
		return networkFile, err
	}

	return networkFile, err
}
