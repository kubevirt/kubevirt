package infraconfigurators

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/client-go/api/v1"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

type PodNetworkInfraConfigurator interface {
	DiscoverPodNetworkInterface(podIfaceName string) error
	PreparePodNetworkInterface() error
	GenerateDomainIfaceSpec() api.Interface
	GenerateDHCPConfig() *cache.DHCPConfig
}

func createAndBindTapToBridge(handler netdriver.NetworkHandler, deviceName string, bridgeIfaceName string, launcherPID int, mtu int, tapOwner string, vmi *v1.VirtualMachineInstance) error {
	err := handler.CreateTapDevice(deviceName, calculateNetworkQueues(vmi), launcherPID, mtu, tapOwner)
	if err != nil {
		return err
	}
	return handler.BindTapDeviceToBridge(deviceName, bridgeIfaceName)
}

func generateTapDeviceName(podInterfaceName string) string {
	return "tap" + podInterfaceName[3:]
}

func validateMTU(mtu int) error {
	if mtu < 0 || mtu > 65535 {
		return fmt.Errorf("MTU value out of range ")
	}
	return nil
}

func calculateNetworkQueues(vmi *v1.VirtualMachineInstance) uint32 {
	if isMultiqueue(vmi) {
		return converter.CalculateNetworkQueues(vmi)
	}
	return 0
}

func isMultiqueue(vmi *v1.VirtualMachineInstance) bool {
	return (vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue != nil) &&
		(*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue)
}
