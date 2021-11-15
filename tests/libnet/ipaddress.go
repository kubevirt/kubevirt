package libnet

import (
	k8sv1 "k8s.io/api/core/v1"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
)

func GetPodIpByFamily(pod *k8sv1.Pod, family k8sv1.IPFamily) string {
	var ips []string
	for _, ip := range pod.Status.PodIPs {
		ips = append(ips, ip.IP)
	}
	return GetIp(ips, family)
}

func GetVmiPrimaryIpByFamily(vmi *v1.VirtualMachineInstance, family k8sv1.IPFamily) string {
	return GetIp(vmi.Status.Interfaces[0].IPs, family)
}

func GetIp(ips []string, family k8sv1.IPFamily) string {
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
