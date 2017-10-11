package main

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ipam"
)

const socketPath = "/run/cni/dhcp.sock"

type NetConf struct {
	types.NetConf
	RealIPAM string `json:"realipam"`
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel, version.All)
}

func loadConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
}

func cmdAdd(args *skel.CmdArgs) error {

	n, cniVersion, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}
	r, err := ipam.ExecAdd(n.RealIPAM, args.StdinData)
	if err != nil {
		return err
	}

	res, err := current.NewResultFromResult(r)
	if err != nil {
		return err
	}
	// remove all routes
	res.Routes = nil
	// don't mess around with existing default gateways
	for _, ipconfig := range res.IPs {
		ipconfig.Gateway = nil
	}

	return types.PrintResult(res, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {

	n, _, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}
	return ipam.ExecDel(n.RealIPAM, args.StdinData)
}
