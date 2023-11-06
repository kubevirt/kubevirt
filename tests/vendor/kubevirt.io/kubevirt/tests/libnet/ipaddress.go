package libnet

import (
	"fmt"
	"net"

	k8sv1 "k8s.io/api/core/v1"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
)

func GetPodIPByFamily(pod *k8sv1.Pod, family k8sv1.IPFamily) string {
	var ips []string
	for _, ip := range pod.Status.PodIPs {
		ips = append(ips, ip.IP)
	}
	return GetIP(ips, family)
}

func GetVmiPrimaryIPByFamily(vmi *v1.VirtualMachineInstance, family k8sv1.IPFamily) string {
	return GetIP(vmi.Status.Interfaces[0].IPs, family)
}

func GetIP(ips []string, family k8sv1.IPFamily) string {
	for _, ip := range ips {
		if family == getFamily(ip) {
			return ip
		}
	}
	return ""
}

func getFamily(ip string) k8sv1.IPFamily {
	if netutils.IsIPv6String(ip) {
		return k8sv1.IPv6Protocol
	}
	return k8sv1.IPv4Protocol
}

func GetLoopbackAddress(family k8sv1.IPFamily) string {
	if family == k8sv1.IPv4Protocol {
		return "127.0.0.1"
	}
	return net.IPv6loopback.String()
}

func GetLoopbackAddressForURL(family k8sv1.IPFamily) string {
	address := GetLoopbackAddress(family)
	if family == k8sv1.IPv6Protocol {
		address = fmt.Sprintf("[%s]", address)
	}
	return address
}

func CidrToIP(cidr string) (string, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}

func FormatIPForURL(ip string) string {
	if netutils.IsIPv6String(ip) {
		return "[" + ip + "]"
	}
	return ip
}
