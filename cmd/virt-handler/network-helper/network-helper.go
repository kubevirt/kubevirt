package main

import (
	"fmt"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/spf13/pflag"

	"encoding/json"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/networking"
)

func main() {
	ip := pflag.String("ip", "", "IP for which to detect the interface for.")
	t := pflag.UintP("target", "t", 1, "Target PID for network namespace")
	pflag.Parse()

	var link netlink.Link
	err := ns.WithNetNSPath(fmt.Sprintf("/proc/%d/ns/net", *t), func(_ ns.NetNS) error {
		var e error
		link, e = networking.GetInterfaceFromIP(*ip)
		return e
	})
	handleErr(err)

	if link == nil {
		handleErr(fmt.Errorf("No device with IP %v found\n", *ip))
	}

	l := networking.Link{Type: link.Type(), IP: *ip, Name: link.Attrs().Name, MAC: link.Attrs().HardwareAddr}

	data, err := json.MarshalIndent(l, "", "  ")
	handleErr(err)
	fmt.Printf("%s\n", string(data))
}

func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
