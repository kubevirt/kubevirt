package netsetup

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

const (
	DefaultTapName      = "kvbpf0"
	DefaultVethLocal    = "kvbpf-veth"
	DefaultVethPeerName = "kvbpf-peer"
)

// EnsureBridgeWiring creates a persistent TAP and a veth pair in the current network namespace.
// BPF attaches to the TAP and to the veth leg named DefaultVethLocal; the peer is available for
// whatever upstream wiring the embedding environment expects.
func EnsureBridgeWiring(tapName, vethLocal, vethPeer string) (tapIdx, vethIdx int, err error) {
	if tapName == "" {
		tapName = DefaultTapName
	}
	if vethLocal == "" {
		vethLocal = DefaultVethLocal
	}
	if vethPeer == "" {
		vethPeer = DefaultVethPeerName
	}

	if err := ensureVeth(vethLocal, vethPeer); err != nil {
		return 0, 0, err
	}
	if err := ensureTAP(tapName); err != nil {
		return 0, 0, err
	}

	tapLink, err := netlink.LinkByName(tapName)
	if err != nil {
		return 0, 0, fmt.Errorf("lookup tap %q: %w", tapName, err)
	}
	vethLink, err := netlink.LinkByName(vethLocal)
	if err != nil {
		return 0, 0, fmt.Errorf("lookup veth %q: %w", vethLocal, err)
	}

	for _, l := range []netlink.Link{tapLink, vethLink} {
		if err := netlink.LinkSetUp(l); err != nil {
			return 0, 0, fmt.Errorf("set %q up: %w", l.Attrs().Name, err)
		}
	}
	peer, err := netlink.LinkByName(vethPeer)
	if err == nil {
		_ = netlink.LinkSetUp(peer)
	}

	return tapLink.Attrs().Index, vethLink.Attrs().Index, nil
}

func ensureVeth(local, peer string) error {
	if _, err := netlink.LinkByName(local); err == nil {
		return nil
	}
	v := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: local},
		PeerName:  peer,
	}
	if err := netlink.LinkAdd(v); err != nil {
		return fmt.Errorf("add veth %s <-> %s: %w", local, peer, err)
	}
	return nil
}

func ensureTAP(name string) error {
	if _, err := netlink.LinkByName(name); err == nil {
		return nil
	}
	la := netlink.NewLinkAttrs()
	la.Name = name
	tap := &netlink.Tuntap{
		Mode:      netlink.TUNTAP_MODE_TAP,
		Flags:     netlink.TUNTAP_DEFAULTS,
		LinkAttrs: la,
	}
	if err := netlink.LinkAdd(tap); err != nil {
		return fmt.Errorf("add tap %q: %w", name, err)
	}
	return nil
}
