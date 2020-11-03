package tests

import (
	"time"

	expect "github.com/google/goexpect"

	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
)

// LoginToCirros call LoggedInFedoraExpecter but does not return the expecter
func LoginToCirros(vmi *v1.VirtualMachineInstance) error {
	expecter, err := LoggedInCirrosExpecter(vmi)
	defer expecter.Close()
	return err
}

// LoggedInCirrosExpecter return prepared and ready to use console expecter for
// Alpine test VM
func LoggedInCirrosExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	expecter, err := console.LoggedInCirrosExpecter(vmi)
	if err != nil {
		return nil, err
	}
	return expecter, configureIPv6OnVMI(vmi, expecter, virtClient)
}

// LoggedInAlpineExpecter return prepared and ready to use console expecter for
// Alpine test VM
func LoggedInAlpineExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	return console.LoggedInAlpineExpecter(vmi)
}

// LoggedInFedoraExpecter return prepared and ready to use console expecter for
// Fedora test VM
func LoggedInFedoraExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	expecter, err := console.LoggedInFedoraExpecter(vmi)
	if err != nil {
		return nil, err
	}
	return expecter, configureIPv6OnVMI(vmi, expecter, virtClient)
}

// ReLoggedInFedoraExpecter return prepared and ready to use console expecter for
// Fedora test VM, when you are reconnecting (no login needed)
func ReLoggedInFedoraExpecter(vmi *v1.VirtualMachineInstance, timeout int) (expect.Expecter, error) {
	return console.ReLoggedInFedoraExpecter(vmi, timeout)
}

func configureIPv6OnVMI(vmi *v1.VirtualMachineInstance, expecter expect.Expecter, virtClient kubecli.KubevirtClient) error {
	hasEth0Iface := func() bool {
		hasNetEth0Batch := append([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "ip a | grep -q eth0; echo $?\n"},
			&expect.BExp{R: console.RetValue("0")}})
		_, err := console.ExpectBatchWithValidatedSend(expecter, hasNetEth0Batch, 30*time.Second)
		return err == nil
	}

	hasGlobalIPv6 := func() bool {
		hasGlobalIPv6Batch := append([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "ip -6 address show dev eth0 scope global | grep -q inet6; echo $?\n"},
			&expect.BExp{R: console.RetValue("0")}})
		_, err := console.ExpectBatchWithValidatedSend(expecter, hasGlobalIPv6Batch, 30*time.Second)
		return err == nil
	}

	clusterSupportsIpv6 := func() bool {
		pod := libvmi.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		for _, ip := range pod.Status.PodIPs {
			if netutils.IsIPv6String(ip.IP) {
				return true
			}
		}
		return false
	}

	if !clusterSupportsIpv6() ||
		(vmi.Spec.Domain.Devices.Interfaces == nil || len(vmi.Spec.Domain.Devices.Interfaces) == 0 || vmi.Spec.Domain.Devices.Interfaces[0].InterfaceBindingMethod.Masquerade == nil) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface != nil && !*vmi.Spec.Domain.Devices.AutoattachPodInterface) ||
		(!hasEth0Iface() || hasGlobalIPv6()) {
		return nil
	}

	addIPv6Address := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "sudo ip -6 addr add fd10:0:2::2/120 dev eth0; echo $?\n"},
		&expect.BExp{R: console.RetValue("0")}})
	resp, err := console.ExpectBatchWithValidatedSend(expecter, addIPv6Address, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6Address failed: %v", resp)
		expecter.Close()
		return err
	}

	time.Sleep(5 * time.Second)
	addIPv6DefaultRoute := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "sudo ip -6 route add default via fd10:0:2::1 src fd10:0:2::2; echo $?\n"},
		&expect.BExp{R: console.RetValue("0")}})
	resp, err = console.ExpectBatchWithValidatedSend(expecter, addIPv6DefaultRoute, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6DefaultRoute failed: %v", resp)
		expecter.Close()
		return err
	}

	return nil
}
