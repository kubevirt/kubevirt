package main

import (
	"fmt"
	"strconv"

	"github.com/songgao/water"
	"github.com/spf13/cobra"
)

func createTapDevice(name string, owner uint, group uint, isMultiqueue bool) error {
	var err error = nil
	config := water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:    name,
			Persist: true,
			Permissions: &water.DevicePermissions{
				Owner: owner,
				Group: group,
			},
			MultiQueue: isMultiqueue,
		},
	}

	_, err = water.New(config)
	return err
}

func NewCreateTapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create-tap",
		Short: "create a tap device in a given PID net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			tapName := cmd.Flag("tap-name").Value.String()
			uidStr := cmd.Flag("uid").Value.String()
			gidStr := cmd.Flag("gid").Value.String()
			isMultiqueueStr := cmd.Flag("multiqueue").Value.String()

			uid, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				return fmt.Errorf("could not parse tap device owner: %v", err)
			}
			gid, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return fmt.Errorf("could not parse tap device group: %v", err)
			}
			isMultiqueue, err := strconv.ParseBool(isMultiqueueStr)
			if err != nil {
				return fmt.Errorf("could not parse multiqueue flag: %v", err)
			}

			if err := createTapDevice(tapName, uint(uid), uint(gid), isMultiqueue); err != nil {
				return fmt.Errorf("failed to create tap device named %s. Reason: %v", tapName, err)
			}

			return nil
		},
	}
}
