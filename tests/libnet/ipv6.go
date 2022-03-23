package libnet

import (
	"time"

	"kubevirt.io/kubevirt/tests/libnet/cluster"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests/console"
)

func configureIPv6OnVMI(vmi *v1.VirtualMachineInstance) error {
	alreadyConfigured := func() bool {
		err := console.RunCommand(vmi, " ip a show lo | grep UP", 30*time.Second)
		return err == nil
	}

	if (vmi.Spec.Domain.Devices.Interfaces == nil || len(vmi.Spec.Domain.Devices.Interfaces) == 0) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface != nil && !*vmi.Spec.Domain.Devices.AutoattachPodInterface) ||
		alreadyConfigured() {
		return nil
	}

	err := console.RunCommand(vmi, "ip link set dev eth0 up", 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("activating eth0 failed: %v", err)
		return err
	}

	err = console.RunCommand(vmi, "ip link set dev lo up", 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("activation lo failed: %v", err)
		return err
	}

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	clusterSupportsIpv4, err := cluster.SupportsIpv4(virtClient)
	if err != nil {
		return err
	}
	if clusterSupportsIpv4 {
		err = console.RunCommand(vmi, "udhcpc", 30*time.Second)
		if err != nil {
			log.DefaultLogger().Object(vmi).Infof("udhcpc failed: %v", err)
			return err
		}
	}

	clusterSupportsIpv6, err := cluster.SupportsIpv6(virtClient)
	if err != nil {
		return err
	}
	if !clusterSupportsIpv6 {
		return nil
	}

	err = console.RunCommand(vmi, "modprobe ipv6", 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("ipv6 activation failed: %v", err)
		return err
	}

	if vmi.Spec.Domain.Devices.Interfaces[0].InterfaceBindingMethod.Masquerade == nil {
		return nil
	}
	err = console.RunCommand(vmi, "ip -6 addr add fd10:0:2::2/120 dev eth0", 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6Address failed: %v", err)
		return err
	}

	time.Sleep(5 * time.Second)

	err = console.RunCommand(vmi, "ip -6 route add default via fd10:0:2::1 src fd10:0:2::2", 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6DefaultRoute failed: %v", err)
		return err
	}

	return nil
}

func WithIPv6(loginTo console.LoginToFunction) console.LoginToFunction {
	return func(vmi *v1.VirtualMachineInstance) error {
		err := loginTo(vmi)
		if err != nil {
			return err
		}
		return configureIPv6OnVMI(vmi)
	}
}
