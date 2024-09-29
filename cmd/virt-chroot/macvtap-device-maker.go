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

package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vishvananda/netlink"
)

func NewCreateMacvtapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create-macvtap",
		Short: "create a macvtap device in a given PID net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := cmd.Flag("name").Value.String()
			lowerDeviceName := cmd.Flag("lower-device-name").Value.String()
			mode := cmd.Flag("mode").Value.String()
			uidStr := cmd.Flag("uid").Value.String()
			uid, err := strconv.Atoi(uidStr)
			if err != nil {
				return fmt.Errorf("could not parse owner: %v", err)
			}
			gidStr := cmd.Flag("gid").Value.String()
			gid, err := strconv.Atoi(gidStr)
			if err != nil {
				return fmt.Errorf("could not parse group: %v", err)
			}
			return createMacvtapDevice(name, lowerDeviceName, mode, uid, gid)
		},
	}
}

func createMacvtapDevice(name, lowerDeviceName, mode string, uid, gid int) error {
	lowerDevice, err := netlink.LinkByName(lowerDeviceName)
	if err != nil {
		return fmt.Errorf("failed to lookup lowerDevice %q: %v", lowerDeviceName, err)
	}

	var macvlanMode netlink.MacvlanMode
	switch mode {
	case "bridge":
		macvlanMode = netlink.MACVLAN_MODE_BRIDGE
	case "vepa":
		macvlanMode = netlink.MACVLAN_MODE_VEPA
	case "private":
		macvlanMode = netlink.MACVLAN_MODE_PRIVATE
	case "passthru":
		macvlanMode = netlink.MACVLAN_MODE_PASSTHRU
	case "source":
		macvlanMode = netlink.MACVLAN_MODE_SOURCE
	default:
		return fmt.Errorf("unknown macvtap mode: %s", mode)
	}

	tapDevice := &netlink.Macvtap{
		Macvlan: netlink.Macvlan{
			LinkAttrs: netlink.LinkAttrs{
				Name:        name,
				ParentIndex: lowerDevice.Attrs().Index,
			},
			Mode: macvlanMode,
		},
	}

	if err := netlink.LinkAdd(tapDevice); err != nil {
		return fmt.Errorf("failed to create macvtap device named %q on %q. Reason: %v", name, lowerDeviceName, err)
	}

	fmt.Printf("Successfully created macvtap device %s", name)
	return nil
}
