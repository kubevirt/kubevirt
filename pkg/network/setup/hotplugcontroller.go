package network

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

type ConcreteController struct {
	ifaceCacheFactory cache.CacheCreator
	vmi               *v1.VirtualMachineInstance
	nsFactory         nsFactory
}

func NewInterfaceController(cacheFactory cache.CacheCreator, ns nsFactory) *ConcreteController {
	return &ConcreteController{
		ifaceCacheFactory: cacheFactory,
		nsFactory:         ns,
	}
}

func (c *ConcreteController) HotplugIface(iface *v1.Network, launcherPID int) error {
	log.Log.V(4).Infof("creating networking infra for network: %s", iface.Name)
	if err := c.nsFactory(launcherPID).Do(func() error {
		return NewVMNetworkConfigurator(
			c.vmi,
			c.ifaceCacheFactory,
		).CreatePodAuxiliaryInfra(launcherPID, iface.Name)
	}); err != nil {
		return fmt.Errorf("setup failed, err: %w", err)
	}

	return nil
}

func (c *ConcreteController) HotplugIfaces(vmi *v1.VirtualMachineInstance, launcherPID int) error {
	c.vmi = vmi

	ifacesToHotplug := InterfacesToHotplug(vmi)
	for i, ifaceToPlug := range ifacesToHotplug {
		log.Log.V(4).Infof("creating networking infra for iface %s", ifaceToPlug.Name)
		if err := c.HotplugIface(&ifacesToHotplug[i], launcherPID); err != nil {
			return fmt.Errorf("error plugging interface [%s]: %w", ifaceToPlug.Name, err)
		}
		log.Log.V(4).Infof("successfully created networking infra for iface: %s", ifaceToPlug.Name)
	}

	// TODO - cleanup binding mechanism resources for unplugged ifaces

	return nil
}

func InterfacesToHotplug(vmi *v1.VirtualMachineInstance) []v1.Network {
	var ifacesToHotplug []v1.Network
	indexedIfacesFromStatus := indexedInterfacesFromStatus(vmi.Status.Interfaces)
	indexedNetsFromSpec := indexedNetworksFromSpec(vmi.Spec.Networks)
	for ifaceName, iface := range indexedNetsFromSpec {
		if _, wasFound := indexedIfacesFromStatus[ifaceName]; !wasFound {
			ifacesToHotplug = append(ifacesToHotplug, iface)
		}
	}
	return ifacesToHotplug
}

func InterfacesToHotUnplug(vmi *v1.VirtualMachineInstance) []v1.VirtualMachineInstanceNetworkInterface {
	var ifacesToUnplug []v1.VirtualMachineInstanceNetworkInterface
	indexedIfacesFromStatus := indexedInterfacesFromStatus(vmi.Status.Interfaces)
	indexedNetsFromSpec := indexedNetworksFromSpec(vmi.Spec.Networks)
	for ifaceName, iface := range indexedIfacesFromStatus {
		if _, wasFound := indexedNetsFromSpec[ifaceName]; !wasFound {
			ifacesToUnplug = append(ifacesToUnplug, iface)
		}
	}
	return ifacesToUnplug
}

func indexedInterfacesFromStatus(interfaces []v1.VirtualMachineInstanceNetworkInterface) map[string]v1.VirtualMachineInstanceNetworkInterface {
	indexedInterfaceStatus := map[string]v1.VirtualMachineInstanceNetworkInterface{}
	for _, iface := range interfaces {
		indexedInterfaceStatus[iface.Name] = iface
	}
	return indexedInterfaceStatus
}

func indexedNetworksFromSpec(networks []v1.Network) map[string]v1.Network {
	indexedNetworks := map[string]v1.Network{}
	for _, network := range networks {
		indexedNetworks[network.Name] = network
	}
	return indexedNetworks
}
