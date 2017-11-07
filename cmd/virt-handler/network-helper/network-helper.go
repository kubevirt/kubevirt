package main

import (
	"fmt"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/spf13/pflag"

	"encoding/json"

	"github.com/vishvananda/netlink"

	"errors"

	"kubevirt.io/kubevirt/pkg/networking"
)

var (
	ErrLinkNotFound = errors.New("Link not found")
)

func main() {
	var ip string
	pflag.StringVar(&ip, "ip", "", "IP for which to detect the interface for.")
	name := pflag.String("name", "", "Name of the interface to detect.")
	t := pflag.UintP("target", "t", 1, "Target PID for network namespace")
	pflag.Parse()

	var link netlink.Link
	if ip != "" {
		err := ns.WithNetNSPath(networking.GetNSFromPID(*t), func(_ ns.NetNS) error {
			var e error
			link, e = networking.GetInterfaceFromIP(ip)
			return e
		})
		handleErr(err)
	} else if *name != "" {
		err := ns.WithNetNSPath(networking.GetNSFromPID(*t), func(_ ns.NetNS) error {
			var e error
			link, e = netlink.LinkByName(*name)
			return e
		})

		if err != nil && err.Error() != "Link not found" {
			handleErr(err)
		}

		if link != nil {
			addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
			handleErr(err)
			if len(addrs) > 0 {
				ip = addrs[0].IP.To4().String()
			}
		}
	}

	// Only return an interface if we found one. Otherwise return nothing
	if link != nil {
		l := networking.Link{Type: link.Type(), IP: ip, Name: link.Attrs().Name, MAC: link.Attrs().HardwareAddr}

		data, err := json.MarshalIndent(l, "", "  ")
		handleErr(err)
		fmt.Printf("%s\n", string(data))
		os.Exit(0)
	}
	// No device found
	fmt.Println("Link not found")
	os.Exit(2)
}

func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
