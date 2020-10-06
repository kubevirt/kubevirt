package libnet

import (
	v1 "k8s.io/api/core/v1"
	netutils "k8s.io/utils/net"
)

func GetPodIpByFamily(pod *v1.Pod, family v1.IPFamily) string {
	var ips []string
	for _, ip := range pod.Status.PodIPs {
		ips = append(ips, ip.IP)
	}
	return getIp(ips, family)
}

func getIp(ips []string, family v1.IPFamily) string {
	for _, ip := range ips {
		if family == getFamily(ip) {
			return ip
		}
	}
	return ""
}

func getFamily(ip string) v1.IPFamily {
	if netutils.IsIPv6String(ip) {
		return v1.IPv6Protocol
	}
	return v1.IPv4Protocol
}
