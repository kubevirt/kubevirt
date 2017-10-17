package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/networking"
)

func main() {
	var hostname string
	var err error

	toolsDir := pflag.String("tools-dir", "/tools", "Location for helper binaries")
	cniConfigDir := pflag.String("cni-config-dir", "/etc/cni/net.d", "Location for CNI configuration")
	cniDir := pflag.String("cni-dir", "/tools/plugins", "Location for CNI plugin binaries")
	pflag.StringVar(&hostname, "hostname-override", "", "Kubernetes Pod to monitor for changes")

	if hostname == "" {
		hostname, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}

	pflag.Parse()
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	networkIntrospector := networking.NewIntrospector(*toolsDir)

	// Create a macvlan device which is attached to the node network
	node, err := virtCli.CoreV1().Nodes().Get(hostname, v1.GetOptions{})
	if err != nil {
		panic(err)
	}

	link, err := networkIntrospector.GetLinkByIP(networking.GetNodeInternalIP(node), 1)
	if err != nil {
		panic(err)
	}

	if err := networking.SetNetConfMaster(*cniConfigDir, "kubevirt.json", link.Name, ""); err != nil {
		panic(err)
	}
	if err := networking.SetNetConfMaster(*cniConfigDir, "nodenetwork.json", "kubevirt0", link.Name); err != nil {
		panic(err)
	}

	cnitool := networking.NewCNITool(*toolsDir, *cniDir, *cniConfigDir)

	// Let's check if we need to adjust the node network
	iface, err := networkIntrospector.GetLinkByName("kubevirt0", 1)
	if err != nil {
		panic(err)
	}

	var doAdd bool
	if iface == nil {
		// No device means that we either deal with a node restart or the first deployment
		// TODO delete all CNI caches
		doAdd = true
	}

	if iface != nil && iface.IP == "" {
		// We have an interface but it has no IP
		// Remove everything and start from scratch
		err := cnitool.CNIDel("kubevirt", "kubevirt", "kubevirt0", nil, 1)
		if err != nil {
			panic(err)
		}
		doAdd = true
	}

	if doAdd {
		res, err := cnitool.CNIAdd("kubevirt", "kubevirt", "kubevirt0", nil, 1)

		if err != nil {
			panic(err)
		}
		fmt.Println(res.String())
	}

	// TODO make sure the the dhcp client is really updating the lease for kubevirt0.
	// TODO take the mac address and send it again to the dhcp client. If the IPs differ, create a new device and move all routes over
	select {}
}
