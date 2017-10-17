package networking

import (
	"net"
	"strings"

	"encoding/json"
	"fmt"
	"os/exec"

	"strconv"

	"crypto/rand"

	"github.com/vishvananda/netlink"
	"k8s.io/api/core/v1"
)

type Link struct {
	Type string           `json:"type"`
	IP   string           `json:"ip"`
	Name string           `json:"name"`
	MAC  net.HardwareAddr `json:"mac"`
}

func GetInterfaceFromIP(ip string) (iface netlink.Link, err error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if ip == strings.Split(addr.String(), "/")[0] {
				return netlink.LinkByName(iface.Name)
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

type IntrospectorInterface interface {
	GetLinkByIP(ip string, pid int) (*Link, error)
}

type introspector struct {
	toolDir string
}

func NewIntrospector(toolDir string) IntrospectorInterface {
	return &introspector{strings.TrimSuffix(toolDir, "/")}
}

func (i *introspector) GetLinkByIP(ip string, pid int) (*Link, error) {
	cmd := exec.Command(i.toolDir+"/network-helper", "--ip", ip, "--target", strconv.Itoa(pid))
	rawLink, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Failed with %v, output: %v", err, string(rawLink))
	}

	link := &Link{}
	err = json.Unmarshal(rawLink, link)
	if err != nil {
		return nil, fmt.Errorf("Could not unmarshal response from network-helper: %v", err)
	}
	return link, nil
}

func GetNSFromPID(pid uint) string {
	return fmt.Sprintf("/proc/%d/ns/net", pid)
}

func RandomMac() (net.HardwareAddr, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	// Set the local bit and don't generate multicast macs
	buf[0] = (buf[0] | 2) & 0xfe
	return buf, nil
}
