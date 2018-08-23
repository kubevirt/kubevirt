package devices

import (
	"bytes"
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type HostBridge struct {
}

func (HostBridge) Setup(vmi *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {
	for i, net := range vmi.Spec.Networks {
		if net.HostBridge != nil {
			podNetNS := podNamespaces.NetNamespace()
			nodeNetNS := hostNamespaces.NetNamespace()

			// Set defaults
			if net.HostBridge.NodeBridgeName == "" {
				net.HostBridge.NodeBridgeName = net.HostBridge.BridgeName
			}

			// First let's create the veth pair and move one part into the host namespace
			// Note: It's important to create the veth pair inside the container, to inherit automatic cleanup for the veth pairs in case of errors.
			linkExist := isLinkExistUnderNS(podNetNS, net.HostBridge.BridgeName)

			if !linkExist {
				_, err := execIPLinkUnderNetNS(
					podNetNS, "add", net.HostBridge.BridgeName, "type", "bridge",
				)
				if err != nil {
					return err
				}
			}

			// Check bridge state under pod namespace
			linkUp, err := isLinkUp(podNetNS, net.HostBridge.BridgeName)
			if err != nil {
				return err
			}
			if linkUp {
				return nil
			}

			// Configure veth under pod namespace
			linkExist = isLinkExistUnderNS(podNetNS, vethName(i))

			peerName := randomPeerName()
			if !linkExist {
				_, err = execIPLinkUnderNetNS(
					podNetNS, "add", vethName(i), "type", "veth", "peer", "name", peerName,
				)
				if err != nil {
					return err
				}

				_, err = execIPLinkUnderNetNS(
					podNetNS, "set", vethName(i), "master", net.HostBridge.BridgeName,
				)
				if err != nil {
					return err
				}

				_, err = execIPLinkUnderNetNS(podNetNS, "set", peerName, "netns", "1")
				if err != nil {
					return err
				}
			}

			// Check bridge state under pod namespace
			linkUp, err = isLinkUp(podNetNS, net.HostBridge.BridgeName)
			if err != nil {
				return err
			}
			if linkUp {
				return nil
			}

			// Connect veth to node bridge
			linkExist = isLinkExistUnderNS(nodeNetNS, net.HostBridge.NodeBridgeName)
			if !linkExist {
				return fmt.Errorf("failed to get bridge %s on the node namespace: %v", net.HostBridge.NodeBridgeName, err)
			}

			_, err = execIPLinkUnderNetNS(nodeNetNS, "show", "type", "bridge", net.HostBridge.NodeBridgeName)
			if err != nil {
				return fmt.Errorf("link %s is not bridge type", net.HostBridge.NodeBridgeName)
			}

			_, err = execIPLinkUnderNetNS(
				nodeNetNS, "set", peerName, "master", net.HostBridge.NodeBridgeName,
			)
			if err != nil {
				return err
			}

			// Set bridge MTU on the veth
			nodeBridgeMTU, err := getLinkMTU(nodeNetNS, net.HostBridge.NodeBridgeName)
			if err != nil {
				return err
			}

			_, err = execIPLinkUnderNetNS(nodeNetNS, "set", peerName, "mtu", nodeBridgeMTU)
			if err != nil {
				return err
			}

			linkUp, err = isLinkUp(nodeNetNS, peerName)
			if err != nil {
				return err
			}
			if !linkUp {
				_, err = execIPLinkUnderNetNS(nodeNetNS, "set", peerName, "up")
				if err != nil {
					return err
				}
			}

			// Do a final configuration under pod namespace
			for _, iface := range []string{vethName(i), net.HostBridge.BridgeName} {

				_, err = execIPLinkUnderNetNS(podNetNS, "set", iface, "mtu", nodeBridgeMTU)
				if err != nil {
					return err
				}

				_, err = execIPLinkUnderNetNS(podNetNS, "set", iface, "up")
				if err != nil {
					return err
				}
			}
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

func execIPLinkUnderNetNS(nsPath string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	args = append([]string{"--net=" + nsPath, "ip", "link"}, args...)
	c := exec.Command("nsenter", args...)
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("%s: %s", string(stderr.Bytes()), err)
	}

	return stdout.Bytes(), nil
}

func isLinkExistUnderNS(nsPath string, linkName string) bool {
	_, err := execIPLinkUnderNetNS(nsPath, "show", linkName)
	if err != nil {
		return false
	}
	return true
}

func isLinkUp(nsPath string, linkName string) (bool, error) {
	// Check bridge state under specified namespace
	nsLink, err := execIPLinkUnderNetNS(nsPath, "show", linkName)
	if err != nil {
		return false, err
	}
	stateRegex := regexp.MustCompile(`state\s(\w+)\s`)
	state := stateRegex.FindStringSubmatch(string(nsLink))
	if state == nil || len(state) < 2 {
		return false, fmt.Errorf("failed to find state stat for the link %s", linkName)
	}

	return state[1] == "UP", nil
}

func getLinkMTU(nsPath string, linkName string) (string, error) {
	// Check bridge state under specified namespace
	nsLink, err := execIPLinkUnderNetNS(nsPath, "show", linkName)
	if err != nil {
		return "", err
	}
	mtuRegex := regexp.MustCompile(`mtu\s(\d+)\s`)
	mtu := mtuRegex.FindStringSubmatch(string(nsLink))
	if mtu == nil || len(mtu) < 2 {
		return "", fmt.Errorf("failed to find state stat for the link %s", linkName)
	}

	return mtu[1], nil
}
