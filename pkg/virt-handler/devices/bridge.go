package devices

import (
	"fmt"
	"math/rand"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/ns"
)

type Bridge struct {
}

func (Bridge) Setup(vmi *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {
	for i, net := range vmi.Spec.Networks {
		if net.Bridge != nil {
			podns, err := ns.GetNS(podNamespaces.NetNamespace())
			if err != nil {
				return err
			}
			nodens, err := ns.GetNS(hostNamespaces.NetNamespace())
			if err != nil {
				return err
			}

			// Set defaults
			if net.Bridge.NodeTargetName == "" && net.Bridge.Type == v1.BridgeOrCreate {
				net.Bridge.NodeTargetName = net.Bridge.TargetName
			}

			// First let's create the veth pair and move one part into the host namespace
			// Note: It's important to create the veth pair inside the container, to inherit automatic cleanup for the veth pairs in case of errors.
			var peerName string
			var bridge netlink.Link
			err = podns.Do(func(_ ns.Namespace) error {
				links, err := netlink.LinkList()
				if err != nil {
					return fmt.Errorf("failed to list interfaces in the container: %v", err)
				}

				// Check if device already exists
				for _, l := range links {
					if l.Attrs().Name == net.Bridge.TargetName {
						bridge = l
						break
					}
				}

				// If device should be brought in by external parties and is there, we are done
				if bridge != nil && net.Bridge.Type == v1.Bridge {
					return nil
				}

				// Create bridge if it does not already exist
				if bridge == nil {
					bridge = &netlink.Bridge{
						LinkAttrs: netlink.LinkAttrs{Name: net.Bridge.TargetName},
					}
					err := netlink.LinkAdd(bridge)
					if err != nil {
						return fmt.Errorf("failed to create bridge %s in the container: %v", net.Bridge.TargetName, err)
					}
					bridge, err = netlink.LinkByName(net.Bridge.TargetName)
					if err != nil {
						return fmt.Errorf("failed to get bridge %s on the node namespace: %v", net.Bridge.NodeTargetName, err)
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
						PeerName: randString(10)}
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
				peerIndex, err := netlink.VethPeerIndex(veth.(*netlink.Veth))
				if err != nil {
					return fmt.Errorf("failed to get peerIndex in the container: %v", err)
				}

				// Get veth peer. If we failed before it might already be moved to another namespace.
				// FIXME here is the only weak point, looks like the netlink library does not properly set
				// the namespace alias if one peer is aleady moved
				peer, err := netlink.LinkByIndex(peerIndex)
				if err != nil {
					// ok, so maybe we moved it already to the host namespaces
					return nodens.Do(func(_ ns.Namespace) error {
						n, err := netlink.LinkByIndex(peerIndex)
						if err != nil {
							return fmt.Errorf("failed searching the peer in node namespace: %v", err)
						}
						peerName = n.Attrs().Name
						return nil
					})
				}

				peerName = peer.Attrs().Name

				// Move veth peer
				err = netlink.LinkSetNsPid(peer, 1)
				if err != nil {
					return fmt.Errorf("failed to move peer to the node namespace: %v", err)
				}

				return nil
			})
			if err != nil {
				return fmt.Errorf("could not setup link %s for network %s: %v", net.Bridge.TargetName, net.Name, err)
			}

			// If we got this far without an error Type "Bridge" should be ready
			if net.Bridge.Type == v1.Bridge {
				return nil
			}

			// If the bridge we create is already up we are done
			if bridge.Attrs().OperState == netlink.OperUp {
				return nil
			}

			// Second let's connect the veth to the host bridge
			var mtu int
			err = nodens.Do(func(_ ns.Namespace) error {
				bridge, err := netlink.LinkByName(net.Bridge.NodeTargetName)
				if err != nil {
					return fmt.Errorf("failed to get bridge %s on the node namespace: %v", net.Bridge.NodeTargetName, err)
				}

				// Check if it is really a bridge
				if _, ok := bridge.(*netlink.Bridge); !ok {
					return fmt.Errorf("link %s is of type %s, expected a bridge", bridge.Attrs().Name, bridge.Type())
				}

				// Get veth peer in this namespace
				peer, err := netlink.LinkByName(peerName)
				if err != nil {
					return fmt.Errorf("failed to get the peer %s in the node namespace: %v", peerName, err)
				}

				// Connect bridge with the peer
				err = netlink.LinkSetMaster(peer, bridge.(*netlink.Bridge))
				if err != nil {
					return fmt.Errorf("failed to connect the peer %s with the bridge %s: %v", peerName, bridge.Attrs().Name, err)
				}

				// Make sure that MTUs match
				if peer.Attrs().MTU != bridge.Attrs().MTU {
					err = netlink.LinkSetMTU(peer, bridge.Attrs().MTU)
					if err != nil {
						return fmt.Errorf("failed to set the peer MTU to the bridges MTU: %v", err)
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

				err = setMTUandUPByName(net.Bridge.TargetName, mtu)
				if err != nil {
					return err
				}

				return nil
			})

			if err != nil {
				return fmt.Errorf("could not finalize link %s for network %s: %v", net.Bridge.TargetName, net.Name, err)
			}

		}
	}
	return nil
}

func vethName(index int) string {
	return fmt.Sprintf("veth%d", index)
}

func (Bridge) Available() error {
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
			return fmt.Errorf("failed to set MTU of link %s to the bridges MTU: %v", name, err)
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
