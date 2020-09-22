package libnet

import (
	v1 "k8s.io/api/core/v1"
	netutils "k8s.io/utils/net"
)

func GetIp(ips []string, family v1.IPFamily) string {
	for _, ip := range ips {
		if netutils.IsIPv6String(ip) {
			if family == v1.IPv6Protocol {
				return ip
			}
		} else {
			if family == v1.IPv4Protocol {
				return ip
			}
		}
	}
	return ""
}

func GetPodIpsStrings(podIPs []v1.PodIP) []string {
	var ips []string
	for _, ip := range podIPs {
		ips = append(ips, ip.IP)
	}
	return ips
}
