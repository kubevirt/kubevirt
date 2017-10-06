package networking

import (
	"net"
	"strings"

	"k8s.io/api/core/v1"
)

func GetInterfaceFromIP(ip string) (iface *net.Interface, err error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return &iface, err
		}
		for _, addr := range addrs {
			if ip == strings.Split(addr.String(), "/")[0] {
				return &iface, nil
			}
		}

	}
	return nil, nil
}

func GetNodeInternalIP(node *v1.Node) (ip string) {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}
