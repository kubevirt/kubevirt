package devices

import (
	"fmt"
	"math/rand"

	"github.com/vishvananda/netlink"

	"os/exec"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type HostBridge struct {
}

func (HostBridge) Setup(vmi *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {
	for i, net := range vmi.Spec.Networks {
		if net.HostBridge != nil {
			c := exec.Command("virt-handler",
				"--create-bridge",
				"--pod-namespace", podNamespaces.NetNamespace(),
				"--host-namespace", hostNamespaces.NetNamespace(),
				"--node-bridge-name", net.HostBridge.NodeBridgeName,
				"--bridge-name", net.HostBridge.BridgeName,
				"--network-name", net.Name,
				fmt.Sprintf("--interface-index=%d", i))

			if output, err := c.CombinedOutput(); err != nil {
				return fmt.Errorf(string(output))
			}

			return nil
		}
	}
	return nil
}

func vethName(index int) string {
	return fmt.Sprintf("k6tveth%d", index)
}

func randomPeerName() string {
	return "k6t" + randString(10)
}

func (HostBridge) Available() error {
	return nil
}

func randString(length int) string {
	b := make([]byte, length)
	letterBytes := "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func setMTUandUPByName(name string, mtu int) error {

	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}

	// Make sure that MTUs match
	if link.Attrs().MTU != mtu {
		err = netlink.LinkSetMTU(link, mtu)
		if err != nil {
			return fmt.Errorf("failed to set MTU of link %s to the bridges MTU %d: %v", name, mtu, err)
		}
	}

	// Bring the link peer in the container up
	if link.Attrs().OperState != netlink.OperUp {
		err = netlink.LinkSetUp(link)
		if err != nil {
			return fmt.Errorf("failed to set the link %s to up: %v", name, err)
		}
	}
	return nil
}

func isNotExist(err error) bool {
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return true
		}
	}
	return false
}
