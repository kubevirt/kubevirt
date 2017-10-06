package main

import (
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/networking"
)

func main() {
	ip := pflag.String("ip", "", "IP for which to detect the interface for.")
	pflag.Parse()

	var hostIf *net.Interface
	err := ns.WithNetNSPath("/proc/1/ns/net", func(_ ns.NetNS) error {
		var e error
		hostIf, e = networking.GetInterfaceFromIP(*ip)
		return e
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if hostIf == nil {
		fmt.Printf("No device with IP %v found\n", *ip)
		os.Exit(1)
	}
	fmt.Println(hostIf.Name)
}
