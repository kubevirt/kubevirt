package devices

import (
	"fmt"
	"math/rand"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/ns"
)

type HostBridge struct {
}

func (HostBridge) Setup(vmi *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {
	for i, net := range vmi.Spec.Networks {
		if net.HostBridge != nil {
			podns, err := ns.GetNS(podNamespaces.NetNamespace())
			if err != nil {
				return err
			}
			nodens, err := ns.GetNS(hostNamespaces.NetNamespace())
			if err != nil {
				return err
			}

			// Set defaults
			if net.HostBridge.NodeBridgeName == "" {
				net.HostBridge.NodeBridgeName = net.HostBridge.BridgeName
			}

			// First let's create the veth pair and move one part into the host namespace
			// Note: It's important to create the veth pair inside the container, to inherit automatic cleanup for the veth pairs in case of errors.
			var peerIndex int
			var bridge netlink.Link
			err = podns.Do(func(_ ns.Namespace) error {
				links, err := netlink.LinkList()
				if err != nil {
					return fmt.Errorf("failed to list interfaces in the container: %v", err)
				}

				// Check if device already exists
				for _, l := range links {
					if l.Attrs().Name == net.HostBridge.BridgeName {
						bridge = l
						break
					}
				}

				// Create bridge if it does not already exist
				if bridge == nil {
					bridge = &netlink.Bridge{
						LinkAttrs: netlink.LinkAttrs{Name: net.HostBridge.BridgeName},
					}
					err := netlink.LinkAdd(bridge)
					if err != nil {
						return fmt.Errorf("failed to create bridge %s in the container: %v", net.HostBridge.BridgeName, err)
					}
					bridge, err = netlink.LinkByName(net.HostBridge.BridgeName)
					if err != nil {
						return fmt.Errorf("failed to get bridge %s on the node namespace: %v", net.HostBridge.NodeBridgeName, err)
					}
				}

				// If the bridge we create is already up we are done
				if bridge.Attrs().OperState == netlink.OperUp {
					return nil
				}

				// Create veth pair if device does not already exists
				veth, err := netlink.LinkByName(vethName(i))
				if err != nil {
					link := &netlink.Veth{
						LinkAttrs: netlink.LinkAttrs{
							Name:        vethName(i),
							MasterIndex: bridge.Attrs().Index,
						},
						PeerName: randomPeerName()}
					err := netlink.LinkAdd(link)
					if err != nil {
						return fmt.Errorf("failed to create veth pair in the container: %v", err)
					}

					// Get device after creation
					veth, err = netlink.LinkByName(vethName(i))
					if err != nil {
						return fmt.Errorf("failed to get veth in the container: %v", err)
					}
				}

				// Check if it is really a veth
				if _, ok := veth.(*netlink.Veth); !ok {
					return fmt.Errorf("link %s is of type %s, expected a veth", veth.Attrs().Name, veth.Type())
				}

				// Get veth peer index
				peerIndex, err = netlink.VethPeerIndex(veth.(*netlink.Veth))
				if err != nil {
					return fmt.Errorf("failed to get peerIndex in the container: %v", err)
				}

				// Get veth peer. If we failed before it might already be moved to another namespace.
				// FIXME here is the only weak point, looks like the netlink library does not properly set
				// the namespace alias if one peer is aleady moved
				peer, err := netlink.LinkByIndex(peerIndex)
				if err != nil {
					// ok, so we must have moved it already
					return nil
				}

				// Check if it is really a veth
				if _, ok := peer.(*netlink.Veth); !ok {
					// ok, so the peer is already moved and we got an index from another namespace
					// this is not even of veth type
					return nil
				}

				// Cross check that this veth is really the expected peer
				vethIndex, err := netlink.VethPeerIndex(peer.(*netlink.Veth))
				if err != nil {
					return fmt.Errorf("failed to get vethIndex for peer in the container: %v", err)
				}

				if vethIndex != veth.Attrs().Index {
					// this is a veth but not the one we expected, we got an index from another namespace
					return nil
				}

				// Move veth peer
				err = netlink.LinkSetNsPid(peer, 1)
				if err != nil {
					return fmt.Errorf("failed to move peer to the node namespace: %v", err)
				}

				// Devices can get a new index after they change their namespace
				peerIndex, err = netlink.VethPeerIndex(veth.(*netlink.Veth))
				if err != nil {
					return fmt.Errorf("failed to get the peer index after the network namespace switch: %v", err)
				}

				return nil
			})
			if err != nil {
				return fmt.Errorf("could not setup link %s for network %s: %v", net.HostBridge.BridgeName, net.Name, err)
			}

			// If the bridge we create is already up we are done
			if bridge.Attrs().OperState == netlink.OperUp {
				return nil
			}

			// Second let's connect the veth to the host bridge
			var mtu int
			err = nodens.Do(func(_ ns.Namespace) error {
				bridge, err := netlink.LinkByName(net.HostBridge.NodeBridgeName)
				if err != nil {
					return fmt.Errorf("failed to get bridge %s on the node namespace: %v", net.HostBridge.NodeBridgeName, err)
				}

				// Check if it is really a bridge
				if _, ok := bridge.(*netlink.Bridge); !ok {
					return fmt.Errorf("link %s is of type %s, expected a bridge", bridge.Attrs().Name, bridge.Type())
				}

				// Get veth peer in this namespace
				peer, err := netlink.LinkByIndex(peerIndex)
				if err != nil {
					return fmt.Errorf("failed to get the peer with index %d in the node namespace: %v", peerIndex, err)
				}

				// Connect bridge with the peer
				err = netlink.LinkSetMaster(peer, bridge.(*netlink.Bridge))
				if err != nil {
					return fmt.Errorf("failed to connect the peer with index %d with the bridge %s: %v", peerIndex, bridge.Attrs().Name, err)
				}

				// Make sure that MTUs match
				if peer.Attrs().MTU != bridge.Attrs().MTU {
					err = netlink.LinkSetMTU(peer, bridge.Attrs().MTU)
					if err != nil {
						return fmt.Errorf("failed to set the peer MTU to the bridges MTU %d: %v", mtu, err)
					}
				}

				// Bring the peer up
				if peer.Attrs().OperState != netlink.OperUp {
					err = netlink.LinkSetUp(peer)
					if err != nil {
						return fmt.Errorf("failed to set the peer in the node namespace to up: %v", err)
					}
				}
				mtu = bridge.Attrs().MTU

				return nil
			})

			if err != nil {
				return fmt.Errorf("could not prepare root namespace part for network %s: %v", net.Name, err)
			}

			// Last let's go back to the container and lets finalize the device configuration there
			err = podns.Do(func(_ ns.Namespace) error {

				err = setMTUandUPByName(vethName(i), mtu)
				if err != nil {
					return err
				}

				err = setMTUandUPByName(net.HostBridge.BridgeName, mtu)
				if err != nil {
					return err
				}

				return nil
			})

			if err != nil {
				return fmt.Errorf("could not finalize link %s for network %s: %v", net.HostBridge.BridgeName, net.Name, err)
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
