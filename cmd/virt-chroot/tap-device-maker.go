package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	retryFlagName       = "retry"
	dummyBridgeFlagName = "dummy-bridge"
)

func createTapDevice(name string, owner uint, group uint, queueNumber int, mtu int) error {
	var err error = nil
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
	if err := netlink.LinkAdd(tapDevice); err != nil {
		return fmt.Errorf("failed to create tap device named %s. Reason: %v", name, err)
	}

	if err := netlink.LinkSetMTU(tapDevice, mtu); err != nil {
		return fmt.Errorf("failed to set MTU on tap device named %s. Reason: %v", name, err)
	}

	return err
}

func addDebugTapFlags(command *cobra.Command) {
	command.Flags().Uint(retryFlagName, 1, "the amount of times the operation is attempted on failure")
	command.Flags().Bool(dummyBridgeFlagName, false, "create and delete a dummy bridge")
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

			err = retryCreateTap(cmd, func() error {
				return withDummyBridgeDevice(cmd, func() error {
					return createTapDevice(tapName, uint(uid), uint(gid), int(queueNumber), int(mtu))
				})
			})

			if err != nil {
				if e := reportLinks(); e != nil {
					return e
				}
				return err
			}

			return nil
		},
	}
}

func reportLinks() error {
	if !envinfo {
		return nil
	}

	links, err := netlink.LinkList()
	if err != nil {
		return err
	}

	linksReport := []string{"Links:"}
	for _, link := range links {
		attrs := link.Attrs()
		linksReport = append(
			linksReport,
			fmt.Sprintf("- %d:\t%-16s[%s]\tm: %d", attrs.Index, attrs.Name, link.Type(), attrs.MasterIndex),
		)
	}
	fmt.Fprintf(os.Stderr, "%s\n", strings.Join(linksReport, "\n"))

	return nil
}

func retryCreateTap(cmd *cobra.Command, f func() error) error {
	retryCount, err := cmd.Flags().GetUint(retryFlagName)
	if err != nil {
		return fmt.Errorf("could not access %s parameter: %v", retryFlagName, err)
	}

	var errorsString []string
	for attemptID := uint(0); attemptID < retryCount; attemptID++ {
		if err := f(); err != nil {
			errorsString = append(errorsString, fmt.Sprintf("[%d]: %v", attemptID, err))
			time.Sleep(time.Second)
		} else {
			fmt.Printf("Operation succeeded [%d]\n", attemptID)
			break
		}
	}
	if len(errorsString) > 0 {
		return fmt.Errorf(strings.Join(errorsString, "\n"))
	}

	return nil
}

func withDummyBridgeDevice(cmd *cobra.Command, f func() error) error {
	enableDummyBridge, err := cmd.Flags().GetBool(dummyBridgeFlagName)
	if err != nil {
		return fmt.Errorf("could not access %s parameter: %v", dummyBridgeFlagName, err)
	}
	if enableDummyBridge {
		const dummyBridgeName = "dummyBridge0"
		if err := createBridgeDevice(dummyBridgeName); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]: %v\n", err)
		} else {
			defer func() {
				if err := deleteBridgeDevice(dummyBridgeName); err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR]: %v\n", err)
				}
			}()
		}
	}

	return f()
}

func createBridgeDevice(name string) error {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: name},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("failed to create bridge %s: %v", name, err)
	}

	return nil
}

func deleteBridgeDevice(name string) error {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: name},
	}

	if err := netlink.LinkDel(bridge); err != nil {
		return fmt.Errorf("failed to delete bridge %s: %v", name, err)
	}

	return nil
}
