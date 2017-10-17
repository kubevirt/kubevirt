package main

import (
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"strings"

	"os"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/vishvananda/netlink"

	"syscall"

	"kubevirt.io/kubevirt/pkg/networking"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

type NetConf struct {
	types.NetConf
	IPAM struct {
		Type string `json:"type,omitempty"`
		Via  string `json:"via,omitempoty"`
	} `json:"ipam,omitempty"`
	Master  string `json:"master"`
	DataDir string `json:"dataDir"`
}

func loadConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}

	if n.Master == "" {
		return nil, "", fmt.Errorf(`"master" field is required. It specifies the host interface name to virtualize`)
	}
	return n, n.CNIVersion, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	n, cniVersion, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	store, err := NewFileStore(n.DataDir, n.Name)
	if err != nil {
		return fmt.Errorf("error creating file based store: %v", err)
	}

	a, err := parseArgs(args.Args)

	if err != nil {
		return err
	}

	var mac net.HardwareAddr

	entry, err := store.Load(args.ContainerID)
	if err != nil {
		return fmt.Errorf("error loading entry from store for id %s: %v", args.ContainerID, err)
	}

	if rawMac := a["mac"]; rawMac != "" {
		mac, err = net.ParseMAC(rawMac)
		if err != nil {
			return fmt.Errorf("error parsing supplied mac address: %v", err)
		}
	} else {
		if entry != nil {
			mac, err = net.ParseMAC(entry.MAC)
			if err != nil {
				return fmt.Errorf("error parsing mac from store for id %s: %v", args.ContainerID, err)
			}
		} else {
			// Generate a mac
			mac, err = networking.RandomMac()
			if err != nil {
				return fmt.Errorf("error generating mac address: %v", err)
			}
		}
		// add mac to env
		envArgs, _ := os.LookupEnv("CNI_ARGS")
		if envArgs != "" && !strings.HasSuffix(envArgs, ";") {
			envArgs = envArgs + ";"
		}
		envArgs = envArgs + fmt.Sprintf("mac=%s", mac.String())
		err = os.Setenv("CNI_ARGS", envArgs)
		if err != nil {
			return fmt.Errorf("error adding mac %s to CNI_ARGS: %v", mac.String(), err)
		}
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

	saved := false
	// Add route over macvtap master interface if it is macvlan
	if n.Master != "" {

		master, err := netlink.LinkByName(n.Master)
		if err != nil {
			return fmt.Errorf("error looking up master device %s: %v", n.Master, err)
		}

		if master.Type() == "macvlan" {
			// First check if we have an IP change. If yes, make sure that we delete the old route
			if entry != nil && !entry.IP.Equal(result.IPs[0].Address.IP) {
				err := deleteRoute(master, entry.IP)
				if err != nil {
					return fmt.Errorf("error removing outdated route: %v", err)
				}
			}

			// Now persist the result, to make sure that we can clean up the following route changes
			err := store.Save(args.ContainerID, result.IPs[0].Address.IP, mac.String())
			if err != nil {
				return fmt.Errorf("error saving entry for id %s in store: %v", args.ContainerID, err)
			}
			saved = true

			// Now create the new route
			err = createRoute(master, result)
			if err != nil {
				return fmt.Errorf("error adding route for %v via %v: %v", result, master.Attrs().Name, err)
			}
			result.Interfaces = append(result.Interfaces, &current.Interface{Name: n.IPAM.Via, Mac: mac.String()})
		} else {
			result.Interfaces = append(result.Interfaces, &current.Interface{Name: n.Master, Mac: mac.String()})
		}
	}

	// If we didn't save the new entry until now, now is the right time
	// Whenever we don't have to modify routes we can just save here since we need no cleanup of old entries
	if !saved {

		// If no macvlan, we just need to store the entry to avoid mac flipping
		err := store.Save(args.ContainerID, result.IPs[0].Address.IP, mac.String())
		if err != nil {
			return fmt.Errorf("error saving entry for id %s in store: %v", args.ContainerID, err)
		}
	}

	return types.PrintResult(result, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	n, _, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	store, err := NewFileStore(n.DataDir, n.Name)
	if err != nil {
		return fmt.Errorf("error creating file based store: %v", err)
	}

	entry, err := store.Load(args.ContainerID)
	if err != nil {
		return fmt.Errorf("error loading entry from store for id %s: %v", args.ContainerID, err)
	}

	err = ipam.ExecDel(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}
	if n.Master != "" {
		master, err := netlink.LinkByName(n.Master)
		if err != nil {
			return fmt.Errorf("error looking up master device %s: %v", n.Master, err)
		}

		if master.Type() == "macvlan" && entry != nil {
			err = deleteRoute(master, entry.IP)
			if err != nil {
				return fmt.Errorf("error removing route for %v; %v ", entry.IP, err)
			}
		}
	}

	err = store.Delete(args.ContainerID)
	if err != nil {
		return fmt.Errorf("error deleting entry for ip %s from store: %v", args.ContainerID, err)
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

	err = netlink.RouteReplace(route)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("error creating route %v: %v", route, err)
	}
	return nil
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
	err = netlink.RouteDel(route)
	// In case that the route does not exist, I got an ESRCH returned
	// TODO should that be added to os.IsNotExist?
	if err != nil && !os.IsNotExist(err) && underlyingError(err) != syscall.ESRCH {
		return fmt.Errorf("error deleting route %v: %v", route, err)
	}
	return nil
}

// underlyingError returns the underlying error for known os error types.
func underlyingError(err error) error {
	switch err := err.(type) {
	case *os.PathError:
		return err.Err
	case *os.LinkError:
		return err.Err
	case *os.SyscallError:
		return err.Err
	}
	return err
}
