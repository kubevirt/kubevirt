/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2024 Red Hat, Inc.
 *
 */

package plugin

import (
	"fmt"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	"log"

	"kubevirt.io/kubevirt/cmd/cniplugins/bridge-binding/pkg/plugin/netlink"
	"kubevirt.io/kubevirt/cmd/cniplugins/bridge-binding/pkg/plugin/sysctl"
)

func CmdAdd(args *skel.CmdArgs) error {
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", netns, err)
	}
	defer netns.Close()

	c := NewCmd(
		netns,
		sysctl.New(),
		netlink.New(netns),
	)

	result, err := c.CmdAddResult(args)
	if err != nil {
		return err
	}
	return result.Print()
}

func CmdDel(args *skel.CmdArgs) error {
	return nil
}

func CmdCheck(args *skel.CmdArgs) error {
	return nil
}

type sysctlAdapter interface {
	IPv4SetUnprivilegedPortStart(int) error
}

type netlinkAdapter interface {
	EnsureBridgeLink() error
	EnsureDummyLink() error
	EnsureTapLink() error
	ConfigurePodNetworks() error
	BridgeInterface() *current.Interface
	DummyInterface() *current.Interface
	TapInterface() *current.Interface
}

type cmd struct {
	netns          ns.NetNS
	sysctlAdapter  sysctlAdapter
	netlinkAdapter netlinkAdapter
}

func NewCmd(netns ns.NetNS, sysctlAdapter sysctlAdapter, netlinkAdapter netlinkAdapter) *cmd {
	return &cmd{
		netns:          netns,
		sysctlAdapter:  sysctlAdapter,
		netlinkAdapter: netlinkAdapter,
	}
}

func (c *cmd) CmdAddResult(args *skel.CmdArgs) (types.Result, error) {
	netConf, cniVersion, err := loadConf(args.StdinData)
	if err != nil {
		return nil, err
	}

	result := current.Result{CNIVersion: cniVersion}

	err = c.netns.Do(func(_ ns.NetNS) error {
		var lerr error

		if lerr = c.sysctlAdapter.IPv4SetUnprivilegedPortStart(0); lerr != nil {
			return lerr
		}

		lerr = c.netlinkAdapter.EnsureBridgeLink()
		if lerr != nil {
			return lerr
		}

		lerr = c.netlinkAdapter.EnsureTapLink()
		if lerr != nil {
			return lerr
		}

		lerr = c.netlinkAdapter.EnsureDummyLink()
		if lerr != nil {
			return lerr
		}

		lerr = c.netlinkAdapter.ConfigurePodNetworks()
		if lerr != nil {
			return lerr
		}

		result.Interfaces = []*current.Interface{
			c.netlinkAdapter.BridgeInterface(),
			c.netlinkAdapter.TapInterface(),
			c.netlinkAdapter.DummyInterface(),
		}

		netname := netConf.Args.Cni.LogicNetworkName
		log.Printf("setup for logical network %s completed successfully", netname)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
