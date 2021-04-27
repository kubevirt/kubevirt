package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func createTapDevice(name string, owner uint, group uint, queueNumber int, mtu int) error {
	tapDevice := &netlink.Tuntap{
		LinkAttrs:  netlink.LinkAttrs{Name: name},
		Mode:       unix.IFF_TAP,
		NonPersist: false,
		Queues:     queueNumber,
		Owner:      uint32(owner),
		Group:      uint32(group),
	}

	// when netlink receives a request for a tap device with 1 queue, it uses
	// the MULTI_QUEUE flag, which differs from libvirt; as such, we need to
	// manually request the single queue flags, enabling libvirt to consume
	// the tap device.
	// See https://github.com/vishvananda/netlink/issues/574
	if queueNumber == 1 {
		tapDevice.Flags = netlink.TUNTAP_DEFAULTS
	}

	// Device creation is retried due to https://bugzilla.redhat.com/1933627
	// which has been observed on multiple occasions on CI runs.
	const retryAttempts = 5
	attempt, err := retry(retryAttempts, func() error {
		return netlink.LinkAdd(tapDevice)
	})
	if err != nil {
		return fmt.Errorf("failed to create tap device named %s. Reason: %v", name, err)
	}

	if err := netlink.LinkSetMTU(tapDevice, mtu); err != nil {
		return fmt.Errorf("failed to set MTU on tap device named %s. Reason: %v", name, err)
	}

	fmt.Printf("Successfully created tap device %s, attempt %d\n", name, attempt)

	return nil
}

func NewCreateTapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create-tap",
		Short: "create a tap device in a given PID net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			tapName := cmd.Flag("tap-name").Value.String()
			uidStr := cmd.Flag("uid").Value.String()
			gidStr := cmd.Flag("gid").Value.String()
			queueNumber, err := cmd.Flags().GetUint32("queue-number")
			if err != nil {
				return fmt.Errorf("could not access queue-number parameter: %v", err)
			}
			mtu, err := cmd.Flags().GetUint32("mtu")
			if err != nil {
				return fmt.Errorf("could not access mtu parameter: %v", err)
			}

			uid, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				return fmt.Errorf("could not parse tap device owner: %v", err)
			}
			gid, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return fmt.Errorf("could not parse tap device group: %v", err)
			}

			return createTapDevice(tapName, uint(uid), uint(gid), int(queueNumber), int(mtu))
		},
	}
}

func retry(retryAttempts uint, f func() error) (uint, error) {
	var errorsString []string
	for attemptID := uint(0); attemptID < retryAttempts; attemptID++ {
		if err := f(); err != nil {
			errorsString = append(errorsString, fmt.Sprintf("[%d]: %v", attemptID, err))
			time.Sleep(time.Second)
		} else {
			if len(errorsString) > 0 {
				fmt.Printf("warning: Tap device creation has been retried: %v", strings.Join(errorsString, "\n"))
			}
			return attemptID, nil
		}
	}

	return retryAttempts, fmt.Errorf(strings.Join(errorsString, "\n"))
}
