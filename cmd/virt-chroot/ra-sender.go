package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/ndp"
)

func NewCreateRADaemonCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ra-sender",
		Short: "reply to RouterSolicitation requests with RouterAdvertisement messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			serverIface := cmd.Flag("listen-on-iface").Value.String()
			ipv6CIDR := cmd.Flag("ipv6-cidr").Value.String()

			err := ndp.SingleClientRouterAdvertisementDaemon(serverIface, ipv6CIDR)
			if err != nil {
				return fmt.Errorf("failed to create the RouterAdvertisement daemon: %v", err)
			}
			return nil
		},
	}
}
