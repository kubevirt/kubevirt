package utils

import (
	"crypto/rand"
	"fmt"
	"net"
	"strings"

	lmf "github.com/subgraph/libmacouflage"

	"kubevirt.io/kubevirt/pkg/log"
)

const tapPrefix = "keth-"

func GetMacDetails(iface string) (net.HardwareAddr, error) {
	currentMac, err := lmf.GetCurrentMac(iface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get mac information for interface: %s", iface)
		return nil, err
	}
	return currentMac, nil
}

func ChangeMacAddr(iface string) (net.HardwareAddr, error) {
	var mac net.HardwareAddr

	currentMac, err := GetMacDetails(iface)
	if err != nil {
		return nil, err
	}

	changed, err := lmf.SpoofMacRandom(iface, false)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to spoof MAC for iface: %s", iface)
		return nil, err
	}

	if changed {
		mac, err = GetMacDetails(iface)
		if err != nil {
			return nil, err
		}
		log.Log.Reason(err).Errorf("Updated Mac for iface: %s - %s", iface, mac)
	}
	return currentMac, nil
}

func GetInterfaceByIP(ip string) (*net.Interface, error) {

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, inter := range interfaces {
		if addrs, err := inter.Addrs(); err == nil {
			for _, addr := range addrs {
				if addr.(*net.IPNet).IP.To4() != nil && strings.Contains(addr.String(), ip) {
					log.Log.Debugf("Found interface: ", inter.Name, " - ", addr.String())
					return &inter, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("failed to find an interface with ip: %s", ip)
}

func GenerateRandomTapName() (string, error) {
	prefix := tapPrefix
	name := make([]byte, 5)
	_, err := rand.Reader.Read(name)
	if err != nil {
		log.Log.Reason(err).Error("failed to generate random interface name")
		return "", err
	}

	return fmt.Sprintf("%s%x", prefix, name), nil
}