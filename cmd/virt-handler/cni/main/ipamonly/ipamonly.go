package main

import (
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/vishvananda/netlink"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

type NetConf struct {
	types.NetConf
	Master string `json:"master"`
}

func loadConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	n, cniVersion, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	r, err := ipam.ExecAdd(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}
	// Convert whatever the IPAM result was into the current Result type
	result, err := current.NewResultFromResult(r)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	// Add route over macvtap master interface
	if n.Master != "" {
		master, err := netlink.LinkByName(n.Master)
		if err != nil {
			return fmt.Errorf("error looking up master device %s: %v", n.Master, err)
		}

		if master.Type() == "macvlan" {
			err := createRoute(master, result)
			if err != nil {
				return fmt.Errorf("error adding route for %v via %v: %v", result, master.Attrs().Name, err)
			}
		}
	}

	return types.PrintResult(result, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	n, _, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}
	a, err := parseArgs(args.Args)

	if err != nil {
		return err
	}
	ip := net.ParseIP(a["ip"])

	err = ipam.ExecDel(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}
	if n.Master != "" {
		master, err := netlink.LinkByName(n.Master)
		if err != nil {
			return fmt.Errorf("error looking up master device %s: %v", n.Master, err)
		}

		if master.Type() == "macvlan" && ip != nil {
			err = deleteRoute(master, ip)
			if err != nil {
				return fmt.Errorf("error removing route for %v; %v ", ip, err)
			}
		}
	}

	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel, version.All)
}

func parseArgs(args string) (map[string]string, error) {
	result := map[string]string{}

	if args == "" {
		return nil, nil
	}

	pairs := strings.Split(args, ";")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("invalid CNI_ARGS pair %q", pair)
		}

		result[strings.ToLower(kv[0])] = kv[1]
	}

	return result, nil
}

func createRoute(dev netlink.Link, result *current.Result) error {

	gw, err := netlink.AddrList(dev, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("error looking up up IP for %s: %v", dev.Attrs().Name, err)
	}
	dst := netlink.NewIPNet(result.IPs[0].Address.IP)
	// Make sure that we exactly match the IP
	dst.Mask = net.IPv4Mask(255, 255, 255, 255)
	route := &netlink.Route{
		Dst: dst,
		Gw:  gw[0].IP,
	}

	return netlink.RouteReplace(route)
}

func deleteRoute(dev netlink.Link, ip net.IP) error {

	gw, err := netlink.AddrList(dev, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("error looking up up IP for %s: %v", dev.Attrs().Name, err)
	}
	dst := netlink.NewIPNet(ip)
	// Make sure that we exactly match the IP
	dst.Mask = net.IPv4Mask(255, 255, 255, 255)
	// remove route
	route := &netlink.Route{
		Dst: dst,
		Gw:  gw[0].IP,
	}
	return netlink.RouteDel(route)
}
