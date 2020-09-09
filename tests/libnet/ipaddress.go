package libnet

import (
	v1 "k8s.io/api/core/v1"
	netutils "k8s.io/utils/net"
)

func GetIp(ips []string, ipv6 bool) string {
	for _, ip := range ips {
		if netutils.IsIPv6String(ip) {
			if ipv6 {
				return ip
			}
		} else {
			if !ipv6 {
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
