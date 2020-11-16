package libnet

import (
	"time"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests/console"
)

func configureIPv6OnVMI(vmi *v1.VirtualMachineInstance) error {

	hasEth0Iface := func() bool {
		err := console.RunCommand(vmi, "ip a | grep -q eth0", 30*time.Second)
		return err == nil
	}

	hasGlobalIPv6 := func() bool {
		err := console.RunCommand(vmi, "ip -6 address show dev eth0 scope global | grep -q inet6", 30*time.Second)
		return err == nil
	}

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	isClusterDualStack, err := IsClusterDualStack(virtClient)
	if err != nil {
		return err
	}

	if !isClusterDualStack ||
		(vmi.Spec.Domain.Devices.Interfaces == nil || len(vmi.Spec.Domain.Devices.Interfaces) == 0 || vmi.Spec.Domain.Devices.Interfaces[0].InterfaceBindingMethod.Masquerade == nil) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface != nil && !*vmi.Spec.Domain.Devices.AutoattachPodInterface) ||
		(!hasEth0Iface() || hasGlobalIPv6()) {
		return nil
	}
	err = console.RunCommand(vmi, "sudo ip -6 addr add fd10:0:2::2/120 dev eth0", 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6Address failed: %v", err)
		return err
	}

	time.Sleep(5 * time.Second)

	err = console.RunCommand(vmi, "sudo ip -6 route add default via fd10:0:2::1 src fd10:0:2::2", 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6DefaultRoute failed: %v", err)
		return err
	}

	return nil
}

func WithIPv6(loginToFactory console.LoginToFactory) console.LoginToFactory {
	return func(vmi *v1.VirtualMachineInstance) error {
		err := loginToFactory(vmi)
		if err != nil {
			return err
		}
		return configureIPv6OnVMI(vmi)
	}
}
