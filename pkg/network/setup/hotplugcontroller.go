package network

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

type ConcreteController struct {
	indexedIfaceStatus map[string]v1.VirtualMachineInstanceNetworkInterface
	ifaceCacheFactory  cache.CacheCreator
	hotpluggedIfaces   []v1.VirtualMachineInstanceNetworkInterface
	removedIfaces      []v1.VirtualMachineInstanceNetworkInterface
	vmi                *v1.VirtualMachineInstance
	nsFactory          nsFactory
}

func NewInterfaceController(cacheFactory cache.CacheCreator, ns nsFactory) *ConcreteController {
	return &ConcreteController{
		ifaceCacheFactory: cacheFactory,
		nsFactory:         ns,
	}
}

func (c *ConcreteController) HotplugIface(iface *v1.VirtualMachineInstanceNetworkInterface, launcherPID int) error {
	log.Log.V(4).Infof("creating networking infra for network: %s, iface %s", iface.Name, iface.InterfaceName)
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
	c.init(vmi)
	for ifaceName, ifaceStatus := range c.indexedIfaceStatus {
		log.Log.V(4).Infof("checking hot-plug status for iface: %s", ifaceName)
		if IsHotplugOperationPending(ifaceStatus) {
			log.Log.Errorf("pod iface %s does not exist yet", ifaceName)
		} else if IsPodNetworkingInfraToBeCreated(ifaceStatus) {
			log.Log.V(4).Infof("creating networking infra for iface %s", ifaceName)
			if err := c.HotplugIface(&ifaceStatus, launcherPID); err != nil {
				c.hotpluggedIfaces = append(c.hotpluggedIfaces, *failedHotplugStatus(ifaceStatus, "failed to create tap + bridge on pod"))
				return fmt.Errorf("error plugging interface [%s]: %w", ifaceName, err)
			}
			c.hotpluggedIfaces = append(c.hotpluggedIfaces, *infraReadyHotplugStatus(ifaceStatus))
			log.Log.V(4).Infof("successfully created networking infra for iface: %s", ifaceName)
			c.indexedIfaceStatus[ifaceName] = *infraReadyHotplugStatus(ifaceStatus)
		} else if ifaceStatus.HotplugInterface != nil && ifaceStatus.HotplugInterface.Type == v1.Unplug {
			log.Log.V(4).Infof("unplug iface: %s", ifaceName)
			c.removedIfaces = append(c.removedIfaces, *unplugStatus(ifaceStatus))
		} else {
			log.Log.V(4).Info("pod networking infra already created, nothing to do here ...")
		}
	}

	c.mergeInterfaceStatus()
	return nil
}

func (c *ConcreteController) init(vmi *v1.VirtualMachineInstance) {
	c.hotpluggedIfaces = nil
	c.removedIfaces = nil

	indexedVMIStatus := map[string]v1.VirtualMachineInstanceNetworkInterface{}
	for i, vmiStatus := range vmi.Status.Interfaces {
		if IsPodNetworkingInfraToBeCreated(vmi.Status.Interfaces[i]) {
			indexedVMIStatus[vmiStatus.Name] = vmiStatus
		}
	}
	c.indexedIfaceStatus = indexedVMIStatus
	c.vmi = vmi
}

func (c *ConcreteController) DynamicIfaceAttachmentStatus() []v1.VirtualMachineInstanceNetworkInterface {
	return append(c.hotpluggedIfaces, c.removedIfaces...)
}

func (c *ConcreteController) mergeInterfaceStatus() {
	for i := range c.vmi.Status.Interfaces {
		iface := c.vmi.Status.Interfaces[i]

		if updatedIface, wasFound := c.indexedIfaceStatus[iface.Name]; wasFound {
			c.vmi.Status.Interfaces[i] = updatedIface
		}
	}
}

func failedHotplugStatus(ifaceStatus v1.VirtualMachineInstanceNetworkInterface, extraMsg string) *v1.VirtualMachineInstanceNetworkInterface {
	newStatus := ifaceStatus.DeepCopy()
	newStatus.HotplugInterface.Phase = v1.InterfaceHotplugPhaseFailed
	newStatus.HotplugInterface.DetailedMessage = extraMsg
	return newStatus
}

func infraReadyHotplugStatus(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) *v1.VirtualMachineInstanceNetworkInterface {
	newStatus := ifaceStatus.DeepCopy()
	newStatus.HotplugInterface.Phase = v1.InterfaceHotplugPhaseInfraReady
	newStatus.HotplugInterface.DetailedMessage = ""
	return newStatus
}

func IsHotplugOperationPending(dynamicIface v1.VirtualMachineInstanceNetworkInterface) bool {
	return dynamicIface.HotplugInterface != nil &&
		dynamicIface.HotplugInterface.Type == v1.Plug &&
		dynamicIface.HotplugInterface.Phase == v1.InterfaceHotplugPhasePending
}

func IsPodNetworkingInfraToBeCreated(dynamicIface v1.VirtualMachineInstanceNetworkInterface) bool {
	return dynamicIface.HotplugInterface != nil &&
		dynamicIface.HotplugInterface.Type == v1.Plug &&
		dynamicIface.HotplugInterface.Phase == v1.InterfaceHotplugPhaseAttachedToPod
}

func IsPodInfraReady(dynamicIface v1.VirtualMachineInstanceNetworkInterface) bool {
	return dynamicIface.HotplugInterface != nil &&
		dynamicIface.HotplugInterface.Type == v1.Plug &&
		(dynamicIface.HotplugInterface.Phase == v1.InterfaceHotplugPhaseInfraReady)
}

func unplugStatus(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) *v1.VirtualMachineInstanceNetworkInterface {
	newStatus := ifaceStatus.DeepCopy()
	newStatus.HotplugInterface.Phase = v1.InterfaceHotplugPhaseInfraReady
	newStatus.HotplugInterface.DetailedMessage = ""
	return newStatus
}
