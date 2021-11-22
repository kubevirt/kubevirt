package network

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	HotplugCmdName   = "addinterface"
	HotUnplugCmdName = "removeinterface"

	ifaceNameArg   = "iface-name"
	networkNameArg = "network-name"
)

var (
	ifaceName   string
	networkName string
	persist     bool
)

type dynamicIfacesCmd struct {
	clientConfig kubecli.KubevirtClient
	isPersistent bool
	namespace    string
}

func NewAddInterfaceCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "addinterface VMI",
		Short:   "add a network interface to a running VM",
		Example: usageAddInterface(),
		Args:    templates.ExactArgs(HotplugCmdName, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newDynamicIfaceCmd(clientConfig, persist)
			if err != nil {
				return fmt.Errorf("error creating the `AddInterface` command: %w", err)
			}
			return c.addInterface(args[0], networkName, ifaceName)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&networkName, networkNameArg, "", "name used in the `networks` section of the spec.")
	_ = cmd.MarkFlagRequired(networkNameArg)
	cmd.Flags().StringVar(&ifaceName, ifaceNameArg, "", "name of the interface being plugged into the VM.")
	_ = cmd.MarkFlagRequired(ifaceNameArg)
	cmd.Flags().BoolVar(&persist, "persist", false, "if set, the added volume will be persisted in the VM spec (if it exists)")

	return cmd
}

func NewRemoveInterfaceCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "removeinterface VMI",
		Short:   "remove a network interface from a running VM",
		Example: usageRemoveInterface(),
		Args:    templates.ExactArgs(HotUnplugCmdName, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newDynamicIfaceCmd(clientConfig, persist)
			if err != nil {
				return fmt.Errorf("error creating the `AddInterface` command: %w", err)
			}
			return c.removeInterface(args[0], networkName, ifaceName)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&networkName, networkNameArg, "", "name used in the `networks` section of the spec.")
	_ = cmd.MarkFlagRequired(networkNameArg)
	cmd.Flags().StringVar(&ifaceName, ifaceNameArg, "", "name of the interface being plugged into the VM.")
	_ = cmd.MarkFlagRequired(ifaceNameArg)
	cmd.Flags().BoolVar(&persist, "persist", false, "if set, the added volume will be persisted in the VM spec (if it exists)")
	return cmd
}

func usageAddInterface() string {
	usage := `  #Dynamically attach a network interface to a running VM.
  {{ProgramName}} addinterface <vm-name> --network-name <net name> --iface-name <iface name>

  #Dynamically attach a network interface to a running VM and persisting it in the VM spec. At next VM restart the network interface will be attached like any other network interface.
  {{ProgramName}} addinterface <vm-name> --network-name <net name> --iface-name <iface name> --persist
  `
	return usage
}

func usageRemoveInterface() string {
	usage := `  #Remove a network interface from a running VM.
  {{ProgramName}} removeinterface fedora-dv --volume-name=example-dv

  #Remove a network interface from a running VM, and persist that change in the VM spec.
  {{ProgramName}} removeinterface <vm-name> --network-name <net name> --iface-name <iface name> --persist
  `
	return usage
}

func newDynamicIfaceCmd(clientCfg clientcmd.ClientConfig, persistState bool) (*dynamicIfacesCmd, error) {
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}
	namespace, _, err := clientCfg.Namespace()
	if err != nil {
		return nil, err
	}
	return &dynamicIfacesCmd{clientConfig: virtClient, isPersistent: persistState, namespace: namespace}, nil
}

func (dic *dynamicIfacesCmd) addInterface(vmName string, networkName string, ifaceName string) error {
	_, _ = fmt.Fprintf(os.Stdout, "add interface for VM [%s]| network name: %s; iface name: %s; persist: %t\n", vmName, networkName, ifaceName, dic.isPersistent)
	if dic.isPersistent {
		return dic.clientConfig.VirtualMachine(dic.namespace).AddInterface(vmName, &v1.AddInterfaceOptions{
			NetworkName:   networkName,
			InterfaceName: ifaceName,
		})
	}
	return dic.clientConfig.VirtualMachineInstance(dic.namespace).AddInterface(vmName, &v1.AddInterfaceOptions{
		NetworkName:   networkName,
		InterfaceName: ifaceName,
	})
}

func (dic *dynamicIfacesCmd) removeInterface(vmName string, networkName string, ifaceName string) error {
	_, _ = fmt.Fprintf(os.Stdout, "remove interface for VM [%s]| network name: %s; iface name: %s; persist: %t\n", vmName, networkName, ifaceName, dic.isPersistent)
	if dic.isPersistent {
		return dic.clientConfig.VirtualMachine(dic.namespace).RemoveInterface(vmName, &v1.RemoveInterfaceOptions{
			NetworkName:   networkName,
			InterfaceName: ifaceName,
		})
	}
	return dic.clientConfig.VirtualMachineInstance(dic.namespace).RemoveInterface(vmName, &v1.RemoveInterfaceOptions{
		NetworkName:   networkName,
		InterfaceName: ifaceName,
	})
}
