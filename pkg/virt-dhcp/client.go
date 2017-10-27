package virtdhcp

import (
	"fmt"
	"net/rpc"

	"kubevirt.io/kubevirt/pkg/log"
)

type VirtDHCPClient struct {
	client *rpc.Client
}

func GetClient() (*VirtDHCPClient, error) {
	conn, err := rpc.Dial("unix", DhcpSocket)
	if err != nil {
		log.Log.Reason(err).Errorf("client failed to connect to virt-dhcp socket: %s", DhcpSocket)
		return nil, err
	}

	return &VirtDHCPClient{client: conn}, nil
}

func (dhcp *VirtDHCPClient) AddIP(mac string, ip string, lease int) error {
	addArgs := &AddArgs{ip, mac, lease}
	addReply := &DhcpReply{}

	dhcp.client.Call("DHCP.AddIP", addArgs, addReply)
	if addReply.Success != true {
		msg := fmt.Sprintf("failed to add interface to dhcp, ip: %s, mac: %s - %s", ip, mac, addReply.Message)
		return fmt.Errorf(msg)
	}
	return nil
}

func (dhcp *VirtDHCPClient) RemoveIP(mac string, ip string) error {
	removeArgs := &RemoveArgs{ip, mac}
	removeReply := &DhcpReply{}

	dhcp.client.Call("DHCP.RemoveIP", removeArgs, removeReply)
	if removeReply.Success != true {
		msg := fmt.Sprintf("failed to remove interface from dhcp, ip: %s, mac: %s - %s", ip, mac, removeReply.Message)
		return fmt.Errorf(msg)
	}
	return nil
}

func (dhcp *VirtDHCPClient) SetIPs(hosts *[]AddArgs, reply *DhcpReply) error {
	addReply := &DhcpReply{}

	dhcp.client.Call("DHCP.SetIPs", hosts, addReply)
	if addReply.Success != true {
		msg := fmt.Sprintf("failed to set known hosts to dhcp: %s", addReply.Message)
		return fmt.Errorf(msg)
	}
	return nil
}
